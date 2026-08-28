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
    avail_gb: float
    swap_gb: float
    os_name: str
    gpu: str
    profile_id: str
    profile: dict

    def summary(self) -> str:
        gpu = self.gpu or "내장 그래픽"
        return (
            f"{self.profile.get('label', self.profile_id)}  "
            f"RAM {self.ram_gb:.0f}GB (가용 {self.avail_gb:.1f}GB"
            + (f", 스왑 {self.swap_gb:.1f}GB" if self.swap_gb >= 0.5 else "")
            + f")  GPU {gpu}  기본 {self.profile.get('pull')}"
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


def avail_ram_gb() -> float:
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
            return 4.0
        return stat.ullAvailPhys / (1024 ** 3)
    if sys.platform == "darwin":
        import re

        out = subprocess.check_output(["vm_stat"], text=True)
        page = 16384
        m = re.search(r"page size of (\d+)", out)
        if m:
            page = int(m.group(1))
        got = {}
        for line in out.splitlines():
            mm = re.match(r'(.+):\s+([\d.]+)', line)
            if mm:
                got[mm.group(1).strip().strip('"')] = float(mm.group(2).rstrip("."))
        pages = (
            got.get("Pages free", 0)
            + got.get("Pages speculative", 0)
            + got.get("Pages purgeable", 0)
        )
        return pages * page / (1024 ** 3)
    try:
        return os.sysconf("SC_AVPHYS_PAGES") * os.sysconf("SC_PAGE_SIZE") / (1024 ** 3)
    except (ValueError, OSError):
        return 4.0


def swap_used_gb() -> float:
    if sys.platform != "darwin":
        return 0.0
    try:
        out = subprocess.check_output(["sysctl", "-n", "vm.swapusage"], text=True)
    except (subprocess.SubprocessError, OSError):
        return 0.0
    import re

    m = re.search(r"used =\s+([\d.]+)([MG])", out)
    if not m:
        return 0.0
    n = float(m.group(1))
    return n / 1024 if m.group(2) == "M" else n


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
    return Machine(
        ram_gb=gb,
        avail_gb=avail_ram_gb(),
        swap_gb=swap_used_gb(),
        os_name=sys.platform,
        gpu=gpu_name(),
        profile_id=pid,
        profile=profile,
    )


def memory_warning(machine: Machine, model: str | None = None) -> str | None:
    name = (model or machine.profile.get("pull") or "").lower()
    need = 8.0 if "14b" in name or "32b" in name else 4.0
    if machine.avail_gb >= need and machine.swap_gb < 2:
        return None
    bits = [
        f"가용 메모리 약 {machine.avail_gb:.1f}GB"
        + (f", 스왑 {machine.swap_gb:.1f}GB" if machine.swap_gb >= 0.5 else "")
        + " 로 빠듯합니다."
    ]
    bits.append("Chrome·Safari 탭을 닫은 뒤 다시 실행하세요. 워드 저장 실패도 대개 이 때문입니다.")
    return " ".join(bits)
