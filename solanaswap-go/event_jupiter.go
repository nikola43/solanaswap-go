package solanaswapgo

import (
	"encoding/json"
	"fmt"

	ag_binary "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/mr-tron/base58"
)

type JupiterSwapEvent struct {
	Amm          solana.PublicKey
	InputMint    solana.PublicKey
	InputAmount  uint64
	OutputMint   solana.PublicKey
	OutputAmount uint64
}

type JupiterFillEvent struct {
	Source         solana.PublicKey
	Destination    solana.PublicKey
	InputAmount    uint64
	OutputAmount   uint64
	InputMint      solana.PublicKey
	OutputMint     solana.PublicKey
	InputDecimals  uint8
	OutputDecimals uint8
}

type JupiterSwapEventData struct {
	JupiterSwapEvent
	InputMintDecimals  uint8
	OutputMintDecimals uint8
}

var JupiterRouteEventDiscriminator = [16]byte{228, 69, 165, 46, 81, 203, 154, 29, 64, 198, 205, 232, 38, 8, 113, 226}
var JupiterFillEventDiscriminator = [8]byte{168, 96, 183, 163, 92, 10, 40, 160}

func (p *Parser) processJupiterSwaps(instructionIndex int) []SwapData {
	var swaps []SwapData
	for _, innerInstructionSet := range p.txMeta.InnerInstructions {
		if innerInstructionSet.Index == uint16(instructionIndex) {
			for _, innerInstruction := range innerInstructionSet.Instructions {

				//if p.isJupiterRouteEventInstruction(innerInstruction) {
				//eventData, err := p.parseJupiterRouteEventInstruction(innerInstruction)

				if p.isJupiterRouteEventInstruction(p.convertRPCToSolanaInstruction(innerInstruction)) {
					eventData, err := p.parseJupiterRouteEventInstruction(p.convertRPCToSolanaInstruction(innerInstruction))
					if err != nil {
						p.Log.Errorf("error processing Jupiter trade event: %s", err)
					}
					if eventData != nil {
						swaps = append(swaps, SwapData{Type: JUPITER, Data: eventData})
					}
				}
			}
		}
	}
	return swaps
}

func (p *Parser) processJupiterFillSwaps(instructionIndex int) []SwapData {
	var swaps []SwapData

	fmt.Println("Processing Jupiter Fill Swaps", p.txMeta.LogMessages)

	for i, innerInstructionSet := range p.txMeta.InnerInstructions {
		_ = i
		for j, innerInstruction := range innerInstructionSet.Instructions {
			_ = j
			decodedBytes, err := base58.Decode(innerInstruction.Data.String())
			if err != nil {
				fmt.Println("error decoding instruction data:", err)
			}
			// fmt.Println("decodedBytes", decodedBytes)

			decodedBytesHex := fmt.Sprintf("%x", decodedBytes)
			fmt.Println("decodedBytesHex", decodedBytesHex)
		}

		// if innerInstructionSet.Index == uint16(instructionIndex) {
		// for _, innerInstruction := range innerInstructionSet.Instructions {
		// 	// fmt.Println("Processing inner instruction:", innerInstruction)
		// 	if p.isJupiterFillEventInstruction(innerInstruction) {
		// 		// eventData, err := p.parseJupiterRouteEventInstruction(innerInstruction)
		// 		// if err != nil {
		// 		// 	p.Log.Errorf("error processing Jupiter trade event: %s", err)
		// 		// }
		// 		// if eventData != nil {
		// 		// 	swaps = append(swaps, SwapData{Type: JUPITER, Data: eventData})
		// 		// }
		// 	}
		// }
		// }
	}

	// for _, innerInstructionSet := range p.txMeta.InnerInstructions {
	// 	fmt.Println("Processing inner instruction set:", len(p.txMeta.InnerInstructions))
	// 	// fmt.Println("Processing inner instruction set:", p.txMeta.LogMessages)
	// 	isJupiterFillEventInstruction := p.isJupiterFillEventInstruction(p.txMeta.LogMessages)

	// 	fmt.Println("isJupiterFillEventInstruction:", isJupiterFillEventInstruction)

	// 	if innerInstructionSet.Index == uint16(instructionIndex) && isJupiterFillEventInstruction {

	// 		eventData, err := p.parseJupiterFillEventInstruction(innerInstructionSet.Instructions)
	// 		if err != nil {
	// 			p.Log.Errorf("error processing Jupiter trade event: %s", err)
	// 		}
	// 		if eventData != nil {
	// 			swaps = append(swaps, SwapData{Type: JUPITER, Data: eventData})
	// 		}

	// 		// for _, innerInstruction := range innerInstructionSet.Instructions {
	// 		// 	fmt.Println("Processing inner instruction:", innerInstruction)

	// 		// 	eventData, err := p.parseJupiterFillEventInstruction(innerInstruction)
	// 		// 	if err != nil {
	// 		// 		p.Log.Errorf("error processing Jupiter trade event: %s", err)
	// 		// 	}
	// 		// 	if eventData != nil {
	// 		// 		swaps = append(swaps, SwapData{Type: JUPITER, Data: eventData})
	// 		// 	}
	// 		// }
	// 	}
	// }
	return swaps
}

// containsDCAProgram checks if the transaction contains the Jupiter DCA program.
func (p *Parser) containsDCAProgram() bool {
	for _, accountKey := range p.allAccountKeys {
		if accountKey.Equals(JUPITER_DCA_PROGRAM_ID) {
			return true
		}
	}
	return false
}

func (p *Parser) parseJupiterRouteEventInstruction(instruction solana.CompiledInstruction) (*JupiterSwapEventData, error) {
	decodedBytes, err := base58.Decode(instruction.Data.String())
	if err != nil {
		return nil, fmt.Errorf("error decoding instruction data: %s", err)
	}
	decoder := ag_binary.NewBorshDecoder(decodedBytes[16:])

	jupSwapEvent, err := handleJupiterRouteEvent(decoder)
	if err != nil {
		return nil, fmt.Errorf("error decoding jupiter swap event: %s", err)
	}

	inputMintDecimals, exists := p.splDecimalsMap[jupSwapEvent.InputMint.String()]
	if !exists {
		inputMintDecimals = 0
	}

	outputMintDecimals, exists := p.splDecimalsMap[jupSwapEvent.OutputMint.String()]
	if !exists {
		outputMintDecimals = 0
	}

	return &JupiterSwapEventData{
		JupiterSwapEvent:   *jupSwapEvent,
		InputMintDecimals:  inputMintDecimals,
		OutputMintDecimals: outputMintDecimals,
	}, nil
}

func (p *Parser) parseJupiterFillEventInstruction(instructions []solana.CompiledInstruction) (*JupiterSwapEventData, error) {

	for _, instruction := range instructions {
		fmt.Println("Processing instruction:", instruction)
	}

	if len(instructions) == 0 {
		return nil, fmt.Errorf("no instructions provided")
	}
	if len(instructions) < 3 {
		return nil, fmt.Errorf("not enough instructions provided")
	}

	systemTransferInstruction := instructions[0]
	// tokenTransferInstruction := instructions[2]

	systemTransferDecodedBytes, err := base58.Decode(systemTransferInstruction.Data.String())
	if err != nil {
		return nil, fmt.Errorf("error decoding system instruction data: %s", err)
	}
	fmt.Println("System Transfer Decoded Bytes:", systemTransferDecodedBytes, len(systemTransferDecodedBytes))
	systemTransferDecoder := ag_binary.NewBorshDecoder(systemTransferDecodedBytes)
	systemTransferEvent, err := handleJupiterFillEvent(systemTransferDecoder)
	if err != nil {
		return nil, fmt.Errorf("error decoding system transfer event: %s", err)
	}
	fmt.Println("System Transfer Event:", systemTransferEvent)

	// tokenTransferDecodedBytes, err := base58.Decode(tokenTransferInstruction.Data.String())
	// if err != nil {
	// 	return nil, fmt.Errorf("error decoding token instruction data: %s", err)
	// }
	// _ = tokenTransferDecodedBytes

	// systemTransferDecoder := ag_binary.NewBorshDecoder(systemTransferDecodedBytes)
	// systemTransferEvent, err := handleJupiterFillEvent(systemTransferDecoder)
	// if err != nil {
	// 	return nil, fmt.Errorf("error decoding system transfer event: %s", err)
	// }
	// fmt.Println("System Transfer Event:", systemTransferEvent)

	// tokenTokenTransferDecoder := ag_binary.NewBorshDecoder(tokenTransferDecodedBytes)
	// tokenTransferEvent, err := handleJupiterFillEvent(tokenTokenTransferDecoder)
	// if err != nil {
	// 	return nil, fmt.Errorf("error decoding token transfer event: %s", err)
	// }
	// fmt.Println("Token Transfer Event:", tokenTransferEvent)

	// tokenTransferDecoder := ag_binary.NewBorshDecoder(tokenTransferDecodedBytes)
	// jupFillSwapEvent, err := handleJupiterRouteEvent(decoder)

	// systemTransfer := ag_binary.NewBorshDecoder(decodedBytes[16:])

	// for _, instruction := range instructions {
	// 	fmt.Println("Processing instruction:", instruction, len(instruction.Data))
	// }

	// decodedBytes, err := base58.Decode(instruction.Data.String())
	// if err != nil {
	// 	return nil, fmt.Errorf("error decoding instruction data: %s", err)
	// }
	// fmt.Println("decodedBytes", decodedBytes)

	return nil, fmt.Errorf("error decoding instruction data")
}

func handleJupiterRouteEvent(decoder *ag_binary.Decoder) (*JupiterSwapEvent, error) {
	var event JupiterSwapEvent
	if err := decoder.Decode(&event); err != nil {
		return nil, fmt.Errorf("error unmarshaling JupiterSwapEvent: %s", err)
	}
	return &event, nil
}

func handleJupiterFillEvent(decoder *ag_binary.Decoder) (*JupiterFillEvent, error) {
	var event JupiterFillEvent
	// var event2 interface{}
	if err := decoder.Decode(&event); err != nil {
		return nil, fmt.Errorf("error unmarshaling JupiterSwapEvent: %s", err)
	}
	fmt.Println("event2", event)
	return &event, nil
}

func (p *Parser) extractSPLDecimals() error {
	mintToDecimals := make(map[string]uint8)

	for _, accountInfo := range p.txMeta.PostTokenBalances {
		if !accountInfo.Mint.IsZero() {
			mintAddress := accountInfo.Mint.String()
			mintToDecimals[mintAddress] = uint8(accountInfo.UiTokenAmount.Decimals)
		}
	}

	processInstruction := func(instr solana.CompiledInstruction) {
		if !p.allAccountKeys[instr.ProgramIDIndex].Equals(solana.TokenProgramID) {
			return
		}

		if len(instr.Data) == 0 || (instr.Data[0] != 3 && instr.Data[0] != 12) {
			return
		}

		if len(instr.Accounts) < 3 {
			return
		}

		mint := p.allAccountKeys[instr.Accounts[1]].String()
		if _, exists := mintToDecimals[mint]; !exists {
			mintToDecimals[mint] = 0
		}
	}

	for _, instr := range p.txInfo.Message.Instructions {
		processInstruction(instr)
	}
	for _, innerSet := range p.txMeta.InnerInstructions {
		for _, instr := range innerSet.Instructions {
			processInstruction(p.convertRPCToSolanaInstruction(instr))
		}
	}

	// Add Native SOL if not present
	if _, exists := mintToDecimals[NATIVE_SOL_MINT_PROGRAM_ID.String()]; !exists {
		mintToDecimals[NATIVE_SOL_MINT_PROGRAM_ID.String()] = 9 // Native SOL has 9 decimal places
	}

	p.splDecimalsMap = mintToDecimals

	return nil
}

// parseJupiterEvents parses Jupiter swap events and returns a SwapInfo representing the entire route
func parseJupiterEvents(events []SwapData) (*SwapInfo, error) {
	if len(events) == 0 {
		return nil, fmt.Errorf("no events provided")
	}

	var firstSwap, lastSwap *JupiterSwapEventData

	for i, event := range events {
		if event.Type != JUPITER {
			continue
		}

		var jupiterEvent JupiterSwapEventData
		eventData, err := json.Marshal(event.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal event data: %v", err)
		}

		if err := json.Unmarshal(eventData, &jupiterEvent); err != nil {
			return nil, fmt.Errorf("failed to unmarshal Jupiter event data: %v", err)
		}

		if i == 0 {
			firstSwap = &jupiterEvent
		}
		lastSwap = &jupiterEvent
	}

	if firstSwap == nil || lastSwap == nil {
		return nil, fmt.Errorf("no valid Jupiter swaps found")
	}

	swapInfo := &SwapInfo{
		AMMs:             []string{string(JUPITER)},
		TokenInMint:      firstSwap.InputMint,
		TokenInAmount:    firstSwap.InputAmount,
		TokenInDecimals:  firstSwap.InputMintDecimals,
		TokenOutMint:     lastSwap.OutputMint,
		TokenOutAmount:   lastSwap.OutputAmount,
		TokenOutDecimals: lastSwap.OutputMintDecimals,
	}

	return swapInfo, nil
}
