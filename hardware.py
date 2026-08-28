"""Pick an Ollama profile from RAM — LG gram 16/32 GB first."""
from __future__ import annotations

import os
import shutil
import subprocess
import sys
from dataclasses import dataclass


@dataclass
class Machine:
    ram_gb: float
    os_name: str
    gpu: str
    profile_id: str
    profile: dict

    def summary(self) -> str:
        return (
            f"{self.profile.get('label', self.profile_id)}  "
            f"RAM {self.ram_gb:.0f} GB  GPU {self.gpu or 'iGPU/CPU'}  "
            f"default {self.profile.get('pull')}"
        )


def ram_gb() -> float:
    if sys.platform == "darwin":
        raw = subprocess.check_output(["sysctl", "-n", "hw.memsize"], text=True).strip()
        return int(raw) / (1024 ** 3)
    if os.name == "nt":
        import ctypes

        class MEMORYSTATUSEX(ctypes.Structure):
            _fields_ = [
                ("dwLength", ctypes.c_ulong),
                ("dwMemoryLoad", ctypes.c_ulong),
                ("ullTotalPhys", ctypes.c_ulonglong),
                ("ullAvailPhys", ctypes.c_ulonglong),
                ("ullTotalPageFile", ctypes.c_ulonglong),
                ("ullAvailPageFile", ctypes.c_ulonglong),
                ("ullTotalVirtual", ctypes.c_ulonglong),
                ("ullAvailVirtual", ctypes.c_ulonglong),
                ("ullAvailExtendedVirtual", ctypes.c_ulonglong),
            ]

        stat = MEMORYSTATUSEX()
        stat.dwLength = ctypes.sizeof(MEMORYSTATUSEX)
        if not ctypes.windll.kernel32.GlobalMemoryStatusEx(ctypes.byref(stat)):
            return 16.0
        return stat.ullTotalPhys / (1024 ** 3)
    try:
        pages = os.sysconf("SC_PHYS_PAGES")
        page = os.sysconf("SC_PAGE_SIZE")
        return (pages * page) / (1024 ** 3)
    except (ValueError, OSError):
        return 16.0


def gpu_name() -> str:
    smi = shutil.which("nvidia-smi")
    if smi:
        try:
            out = subprocess.check_output(
                [smi, "--query-gpu=name,memory.total", "--format=csv,noheader"],
                text=True,
                timeout=4,
                stderr=subprocess.DEVNULL,
            ).strip()
            if out:
                return out.splitlines()[0].strip()
        except (subprocess.SubprocessError, OSError):
            pass
    if os.name == "nt":
        try:
            out = subprocess.check_output(
                ["wmic", "path", "win32_VideoController", "get", "Name"],
                text=True,
                timeout=6,
                stderr=subprocess.DEVNULL,
            )
            names = [ln.strip() for ln in out.splitlines() if ln.strip() and ln.strip() != "Name"]
            if names:
                return names[0]
        except (subprocess.SubprocessError, OSError, FileNotFoundError):
            pass
    if sys.platform == "darwin":
        return "Apple Silicon"
    return ""


def pick_profile(
    cfg: dict,
    ram: float | None = None,
    explicit: str | None = None,
    platform: str | None = None,
) -> tuple[str, dict]:
    profiles = cfg.get("profiles") or {}
    if explicit:
        if explicit not in profiles:
            known = ", ".join(profiles) or "(none)"
            raise KeyError(f"unknown profile {explicit}. known: {known}")
        return explicit, profiles[explicit]
    gb = ram if ram is not None else ram_gb()
    plat = platform if platform is not None else sys.platform
    if plat == "darwin" and 20 <= gb < 28 and "mac24" in profiles:
        return "mac24", profiles["mac24"]
    ordered = [
        ("gram16", profiles.get("gram16") or {}),
        ("gram32", profiles.get("gram32") or {}),
        ("mac24", profiles.get("mac24") or {}),
    ]
    for pid, p in ordered:
        if not p:
            continue
        lo = float(p.get("ram_min", 0))
        hi = float(p.get("ram_max", 999))
        if lo <= gb < hi:
            return pid, p
    fallback = profiles.get("gram16") or {"pull": "qwen3:8b", "model_pref": ["qwen3:8b"]}
    return "gram16", fallback


def inspect(cfg: dict, explicit: str | None = None) -> Machine:
    gb = ram_gb()
    pid, profile = pick_profile(cfg, gb, explicit)
    return Machine(ram_gb=gb, os_name=sys.platform, gpu=gpu_name(), profile_id=pid, profile=profile)
