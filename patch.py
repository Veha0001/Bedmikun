from demodapk.hex import patch_codes
from shutil import copy

TARGET = "./Minecraft.Windows.exe"
SIGNATURE: dict = {
    "x64": [
        "15 B0 01 48 8B 4C ?? ?? 48 33 ?? ?? ?? ?? ?? ?? 48 83 C4 40 5B C3 48 8B ?? ?? ?? ?? ?? 48 89 | 15 B0 00",
        "84 C0 74 23 48 83 C3 10 48 3B DF 75 E3 B0 01 48 | 84 C0 74 23 48 83 C3 10 48 3B DF 75 E3 B0 00"
    ]
}
copy(src=TARGET, dst=TARGET+".bak")
patch_codes(src=TARGET, codes=SIGNATURE.get("x64", []), verbose=True)
