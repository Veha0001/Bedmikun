# Minecraft.Windows.exe

## Signatures (x64)

These signatures can be used to find the `is_trial` function.

```asm
10 84 ?? ?? 15 B0 01 48 8B 4C ?? ?? 48 33 ?? ?? ?? ?? ?? ?? 48 83 C4 40 5B C3 48 8B ?? ?? ?? ?? 48 89
```

> Replace: `B0 01` to `B0 00`


```asm
84 C0 74 23 48 83 C3 10 48 3B DF 75 E3 B0 01 48
```

> Replace: `B0 01` to `B0 00`

### Manual Patching with IDA

This method requires fully unlocking access permission to the `Minecraft.Windows.exe` file.

1.  Load the executable in IDA.
2.  Go to `Search > Sequence of bytes...` (or press `Alt+B`).
3.  In the "Sequence of bytes" window, enter the following pattern:
    `40 ? 48 83 EC ? 48 8B ? ? ? ? ? 48 33 ? 48 89 ? ? ? 48 8B ? 48 8B ? ? 48 8B ? 48 8B ? ? ? ? ? FF 15`
4.  Once IDA finds the location, press `F5` to decompile the function.
5.  Look for a line that says `return 1;`.
6.  Click on that line to go to the corresponding assembly instruction.
7.  To change the return value:
    *   Go to `Edit > Patch program > Assemble...` (or press `Spacebar` on the instruction).
    *   Change the instruction to `mov eax, 0` and click OK.
8.  To save the changes to the executable:
    *   Go to `Edit > Patch program > Apply patches to input file...`.
    *   This will create a patched version of the executable.
