# evm-golang

A Simple evm in golang

## description

This a simple ethereum virtual machine. Given a simple contract in bytecode format, this evm will execute the contract. A compiler can be created to compile solidity or hyper to the bytecode.

## usage

The program comes with a bytecode already hard-coded that you can run and confirm whether it works. The example bytecode is found in the main function. You can try providing different bytecodes.

```bash
go run main.go
```

The example bytecode provided by the program

```go
code := []byte{
		0x60, 0x0a, // PUSH1 0x0a
		0x60, 0x14, // PUSH1 0x14
		0x01, // ADD
		0x00, // STOP
	}
```

When you run the program, you will see the following output. The printing of the current executing opcode is for debug purposes. Normally, the program will execute the whole contract and stop without having to display the currently executing opcode.

```plaintext
96
96
1
0
Error: STOP
EVM Execution Complete
```

**understanding the output**

The opcodes are in hex format and as a result the 96 you are seeing in the output is the decimal conversion of the currently executing instruction. Do not worry about the "Error: STOP", as it does not really indicate an error in this case. In other programs, given the wrong opcode, it will indeed indicate an error.

## opcodes

These are the accepted opcodes

```plaintext
0x00 - STOP
0x01 - ADD
0x02 - MUL
0x03 - SUB
0x04 - DIV
0x10 - LT
0x11 - GT
0x14 - EQ
0x54 - SLOAD
0x55 - SSTORE
0x56 - JUMP
0x57 - JUMPI
0x60 - PUSH1
0x80 - DUP1
0x90 - SWAP1
0xa0 - LOG0
0xf0 - CREATE
0xf1 - CALL
0xf3 - RETURN
0xfd - REVERT
```

You can try playing around with a few programs and see how it goes...

## contributors

[nutcas](https://github.com/nutcas3)
