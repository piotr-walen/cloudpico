### Prerequisites

#### 1) Install `usbipd-win` and attach the Pico to WSL

Install **usbipd-win** (Microsoft docs):
[https://learn.microsoft.com/en-us/windows/wsl/connect-usb](https://learn.microsoft.com/en-us/windows/wsl/connect-usb)

In an **Administrator PowerShell**, bind the device once and attach it to WSL:

```powershell
usbipd list
usbipd bind --busid <BUSID>          # run once (requires admin)
usbipd attach --wsl --busid <BUSID>  # run each time you want it in WSL
```

#### 2) Install TinyGo

Install **TinyGo**:
[https://tinygo.org/getting-started/install/](https://tinygo.org/getting-started/install/)

---

### Flashing the Pico

1. Put the board into **BOOTSEL** mode (it should appear as a USB drive in Windows).

2. Flash using tinygo flash 

```bash
tinygo flash -target=pico2-w .
```

OR
2. Build the UF2 binary:

```bash
tinygo build -target=pico2-w -o main.uf2 .
```

```bash
cp main.uf2 /mnt/<pico_drive_letter>/
```

### Serial monitoring

```bash
tinygo monitor
```
