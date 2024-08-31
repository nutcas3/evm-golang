package main

import (
	"fmt"
	"math/big"
	"crypto/sha256"
)
const (
	MaxStackDepth = 1024
	MaxMemorySize = 1 << 25 // 32 MB
)

// DataType represents different Ethereum data types
type DataType int

const (
	Uint256 DataType = iota
	Address
	Bytes32
)

// Value represents a typed value in the EVM
type Value struct {
	Type  DataType
	Value *big.Int
}

// Stack represents the EVM stack
type Stack struct {
	data []*Value
}

// Memory represents the EVM memory
type Memory struct {
	data []byte
}

// Storage represents the contract storage
type Storage map[string]*Value

// Log represents an event log
type Log struct {
	Address [20]byte
	Topics  [][32]byte
	Data    []byte
}

// Context represents the execution context
type Context struct {
	BlockNumber *big.Int
	Timestamp   *big.Int
	Sender      [20]byte
	GasLimit    uint64
	GasPrice    *big.Int
}

// Contract represents a smart contract
type Contract struct {
	Address [20]byte
	Code    []byte
	Storage Storage
}

// EVM represents the Ethereum Virtual Machine
type EVM struct {
	stack      *Stack
	memory     *Memory
	contract   *Contract
	pc         uint64 // Program Counter
	gas        uint64
	context    *Context
	contracts  map[[20]byte]*Contract
	returnData []byte
	logs       []Log
	depth      int
}

// NewEVM creates a new instance of EVM
func NewEVM(context *Context) *EVM {
	return &EVM{
		stack:     &Stack{},
		memory:    &Memory{},
		pc:        0,
		gas:       context.GasLimit,
		context:   context,
		contracts: make(map[[20]byte]*Contract),
		depth:     0,
	}
}

// Stack methods
func (s *Stack) push(value *Value) error {
	if len(s.data) >= MaxStackDepth {
		return fmt.Errorf("stack overflow")
	}
	s.data = append(s.data, value)
	return nil
}

func (s *Stack) pop() (*Value, error) {
	if len(s.data) == 0 {
		return nil, fmt.Errorf("stack underflow")
	}
	value := s.data[len(s.data)-1]
	s.data = s.data[:len(s.data)-1]
	return value, nil
}

// Memory methods
func (m *Memory) store(offset uint64, value []byte) error {
	if offset+uint64(len(value)) > MaxMemorySize {
		return fmt.Errorf("memory size exceeded")
	}
	if uint64(len(m.data)) < offset+uint64(len(value)) {
		newSize := offset + uint64(len(value))
		newData := make([]byte, newSize)
		copy(newData, m.data)
		m.data = newData
	}
	copy(m.data[offset:], value)
	return nil
}

func (m *Memory) load(offset uint64, size uint64) ([]byte, error) {
	if offset+size > uint64(len(m.data)) {
		return nil, fmt.Errorf("memory access out of bounds")
	}
	return m.data[offset : offset+size], nil
}

// ExecuteOpcode executes a single opcode
func (evm *EVM) ExecuteOpcode(opcode byte) error {
	switch opcode {
	case 0x00: // STOP
		return fmt.Errorf("STOP")
	case 0x01: // ADD
		return evm.binaryOperation(func(a, b *big.Int) *big.Int { return new(big.Int).Add(a, b) }, 3)
	case 0x02: // MUL
		return evm.binaryOperation(func(a, b *big.Int) *big.Int { return new(big.Int).Mul(a, b) }, 5)
	case 0x03: // SUB
		return evm.binaryOperation(func(a, b *big.Int) *big.Int { return new(big.Int).Sub(a, b) }, 3)
	case 0x04: // DIV
		return evm.binaryOperation(func(a, b *big.Int) *big.Int {
			if b.Sign() == 0 {
				return new(big.Int)
			}
			return new(big.Int).Div(a, b)
		}, 5)
	case 0x10: // LT
		return evm.compareOperation(func(a, b *big.Int) bool { return a.Cmp(b) < 0 }, 3)
	case 0x11: // GT
		return evm.compareOperation(func(a, b *big.Int) bool { return a.Cmp(b) > 0 }, 3)
	case 0x14: // EQ
		return evm.compareOperation(func(a, b *big.Int) bool { return a.Cmp(b) == 0 }, 3)
	case 0x54: // SLOAD
		return evm.sload(200)
	case 0x55: // SSTORE
		return evm.sstore(20000)
	case 0x56: // JUMP
		return evm.jump(8)
	case 0x57: // JUMPI
		return evm.jumpi(10)
	case 0x60: // PUSH1
		return evm.push(1, 3)
	case 0x80: // DUP1
		return evm.dup(1, 3)
	case 0x90: // SWAP1
		return evm.swap(1, 3)
	case 0xa0: // LOG0
		return evm.log(0, 375)
	case 0xf0: // CREATE
		return evm.create(32000)
	case 0xf1: // CALL
		return evm.call(40)
	case 0xf3: // RETURN
		return evm.returnOp(0)
	case 0xfd: // REVERT
		return evm.revert(0)
	default:
		return fmt.Errorf("unknown opcode: 0x%x", opcode)
	}
}

func (evm *EVM) binaryOperation(op func(*big.Int, *big.Int) *big.Int, gasCost uint64) error {
	if err := evm.useGas(gasCost); err != nil {
		return err
	}
	b, err := evm.stack.pop()
	if err != nil {
		return err
	}
	a, err := evm.stack.pop()
	if err != nil {
		return err
	}
	result := op(a.Value, b.Value)
	return evm.stack.push(&Value{Type: Uint256, Value: result})
}

func (evm *EVM) compareOperation(op func(*big.Int, *big.Int) bool, gasCost uint64) error {
	if err := evm.useGas(gasCost); err != nil {
		return err
	}
	b, err := evm.stack.pop()
	if err != nil {
		return err
	}
	a, err := evm.stack.pop()
	if err != nil {
		return err
	}
	result := op(a.Value, b.Value)
	if result {
		return evm.stack.push(&Value{Type: Uint256, Value: big.NewInt(1)})
	}
	return evm.stack.push(&Value{Type: Uint256, Value: big.NewInt(0)})
}

func (evm *EVM) sload(gasCost uint64) error {
	if err := evm.useGas(gasCost); err != nil {
		return err
	}
	key, err := evm.stack.pop()
	if err != nil {
		return err
	}
	value := evm.contract.Storage[key.Value.String()]
	if value == nil {
		value = &Value{Type: Uint256, Value: big.NewInt(0)}
	}
	return evm.stack.push(value)
}

func (evm *EVM) sstore(gasCost uint64) error {
	if err := evm.useGas(gasCost); err != nil {
		return err
	}
	value, err := evm.stack.pop()
	if err != nil {
		return err
	}
	key, err := evm.stack.pop()
	if err != nil {
		return err
	}
	evm.contract.Storage[key.Value.String()] = value
	return nil
}

func (evm *EVM) jump(gasCost uint64) error {
	if err := evm.useGas(gasCost); err != nil {
		return err
	}
	dest, err := evm.stack.pop()
	if err != nil {
		return err
	}
	evm.pc = dest.Value.Uint64() - 1 // -1 because pc will be incremented after this
	return nil
}

func (evm *EVM) jumpi(gasCost uint64) error {
	if err := evm.useGas(gasCost); err != nil {
		return err
	}
	condition, err := evm.stack.pop()
	if err != nil {
		return err
	}
	dest, err := evm.stack.pop()
	if err != nil {
		return err
	}
	if condition.Value.Sign() != 0 {
		evm.pc = dest.Value.Uint64() - 1 // -1 because pc will be incremented after this
	}
	return nil
}

func (evm *EVM) push(size uint64, gasCost uint64) error {
	if err := evm.useGas(gasCost); err != nil {
		return err
	}
	if evm.pc+1+size > uint64(len(evm.contract.Code)) {
		return fmt.Errorf("push: out of bounds")
	}
	value := new(big.Int).SetBytes(evm.contract.Code[evm.pc+1 : evm.pc+1+size])
	evm.pc += size
	return evm.stack.push(&Value{Type: Uint256, Value: value})
}

func (evm *EVM) dup(pos uint64, gasCost uint64) error {
	if err := evm.useGas(gasCost); err != nil {
		return err
	}
	if uint64(len(evm.stack.data)) < pos {
		return fmt.Errorf("dup: stack underflow")
	}
	return evm.stack.push(evm.stack.data[uint64(len(evm.stack.data))-pos])
}

func (evm *EVM) swap(pos uint64, gasCost uint64) error {
	if err := evm.useGas(gasCost); err != nil {
		return err
	}
	if uint64(len(evm.stack.data)) <= pos {
		return fmt.Errorf("swap: stack underflow")
	}
	evm.stack.data[uint64(len(evm.stack.data))-1], evm.stack.data[uint64(len(evm.stack.data))-1-pos] =
		evm.stack.data[uint64(len(evm.stack.data))-1-pos], evm.stack.data[uint64(len(evm.stack.data))-1]
	return nil
}

func (evm *EVM) log(topicCount uint64, gasCost uint64) error {
	if err := evm.useGas(gasCost); err != nil {
		return err
	}
	size, err := evm.stack.pop()
	if err != nil {
		return err
	}
	offset, err := evm.stack.pop()
	if err != nil {
		return err
	}
	data, err := evm.memory.load(offset.Value.Uint64(), size.Value.Uint64())
	if err != nil {
		return err
	}
	topics := make([][32]byte, topicCount)
	for i := uint64(0); i < topicCount; i++ {
		topic, err := evm.stack.pop()
		if err != nil {
			return err
		}
		copy(topics[i][:], topic.Value.Bytes())
	}
	log := Log{
		Address: evm.contract.Address,
		Topics:  topics,
		Data:    data,
	}
	evm.logs = append(evm.logs, log)
	return nil
}

func (evm *EVM) create(gasCost uint64) error {
	if err := evm.useGas(gasCost); err != nil {
		return err
	}
	size, err := evm.stack.pop()
	if err != nil {
		return err
	}
	offset, err := evm.stack.pop()
	if err != nil {
		return err
	}
	value, err := evm.stack.pop()
	if err != nil {
		return err
	}
	code, err := evm.memory.load(offset.Value.Uint64(), size.Value.Uint64())
	if err != nil {
		return err
	}
	address := evm.createAddress(evm.contract.Address, uint64(len(evm.contracts)))
	contract := &Contract{
		Address: address,
		Code:    code,
		Storage: make(Storage),
	}
	evm.contracts[address] = contract
	return evm.stack.push(&Value{Type: Address, Value: new(big.Int).SetBytes(address[:])})
}

 func (evm *EVM) call(gasCost uint64) error {
	if err := evm.useGas(gasCost); err != nil {
		return err
	}
	// Pop arguments from stack
	argsSize, err := evm.stack.pop()
	if err != nil {
		return err
	}
	argsOffset, err := evm.stack.pop()
	if err != nil {
		return err
	}
	retSize, err := evm.stack.pop()
	if err != nil {
		return err
	}
	retOffset, err := evm.stack.pop()
	if err != nil {
		return err
	}
	value, err := evm.stack.pop()
	if err != nil {
		return err
	}
	address, err := evm.stack.pop()
	if err != nil {
		return err
	}
	gasLimit, err := evm.stack.pop()
	if err != nil {
		return err
	}

	// Load call data from memory
	args, err := evm.memory.load(argsOffset.Value.Uint64(), argsSize.Value.Uint64())
	if err != nil {
		return err
	}

	// Get the contract to call
	// Get the contract to call
	var contract *Contract
	if addr, ok := address.Value.(*big.Int); ok {
		var contractAddress [20]byte
		copy(contractAddress[:], addr.Bytes())
		contract = evm.contracts[contractAddress]
	}
	if contract == nil {
		return fmt.Errorf("contract not found")
	}

	// Execute the code of the called contract
	calleeEVM := &EVM{
		stack:     &Stack{},
		memory:    &Memory{},
		contract:  contract,
		pc:        0,
		gas:       gasLimit.Uint64(),
		context:   evm.context,
		contracts: evm.contracts,
		depth:     evm.depth + 1,
	}

	// Run the callee contract's code
	for calleeEVM.pc < uint64(len(contract.Code)) {
		if err := calleeEVM.ExecuteOpcode(contract.Code[calleeEVM.pc]); err != nil {
			return err
		}
	}

	// Store the return data
	returnDataSize := retSize.Value.Uint64()
	returnData, err := calleeEVM.memory.load(retOffset.Value.Uint64(), returnDataSize)
	if err != nil {
		return err
	}
	evm.returnData = returnData
	return nil
}

func (evm *EVM) returnOp(gasCost uint64) error {
	if err := evm.useGas(gasCost); err != nil {
		return err
	}
	size, err := evm.stack.pop()
	if err != nil {
		return err
	}
	offset, err := evm.stack.pop()
	if err != nil {
		return err
	}
	data, err := evm.memory.load(offset.Value.Uint64(), size.Value.Uint64())
	if err != nil {
		return err
	}
	evm.returnData = data
	return nil
}

func (evm *EVM) revert(gasCost uint64) error {
	if err := evm.useGas(gasCost); err != nil {
		return err
	}
	size, err := evm.stack.pop()
	if err != nil {
		return err
	}
	offset, err := evm.stack.pop()
	if err != nil {
		return err
	}
	data, err := evm.memory.load(offset.Value.Uint64(), size.Value.Uint64())
	if err != nil {
		return err
	}
	evm.returnData = data
	return fmt.Errorf("revert with data")
}

func (evm *EVM) useGas(cost uint64) error {
	if evm.gas < cost {
		return fmt.Errorf("out of gas")
	}
	evm.gas -= cost
	return nil
}

func (evm *EVM) createAddress(callerAddress [20]byte, nonce uint64) [20]byte {
	var address [20]byte
	copy(address[:], sha256.New().Sum(nil)) // Placeholder, use proper address calculation
	return address
}

func main() {
	context := &Context{
		BlockNumber: big.NewInt(1),
		Timestamp:   big.NewInt(1),
		Sender:      [20]byte{},
		GasLimit:    1000000,
		GasPrice:    big.NewInt(1),
	}

	evm := NewEVM(context)

	code := []byte{
		0x60, 0x0a, // PUSH1 0x0a
		0x60, 0x14, // PUSH1 0x14
		0x01,       // ADD
		0x00,       // STOP
	}

	contract := &Contract{
		Address: [20]byte{},
		Code:    code,
		Storage: make(Storage),
	}

	evm.contract = contract

	for evm.pc < uint64(len(contract.Code)) {
		if err := evm.ExecuteOpcode(contract.Code[evm.pc]); err != nil {
			fmt.Println("Error:", err)
			break
		}
		evm.pc++
	}

	fmt.Println("EVM Execution Complete")
}
