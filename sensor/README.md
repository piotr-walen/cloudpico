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

2. Build the UF2 binary:

```bash
tinygo build -target=pico2-w -o main.uf2 .
```

3. Mount the Pico drive inside WSL (replace `<pico_drive_letter>` with the Windows drive letter, e.g. `E`):

```bash
sudo mkdir -p /mnt/<pico_drive_letter>
sudo mount -t drvfs <pico_drive_letter>: /mnt/<pico_drive_letter> \
  -o uid=$(id -u),gid=$(id -g),metadata
```

4. Copy the UF2 to the mounted drive:

```bash
cp main.uf2 /mnt/<pico_drive_letter>/
```

After the copy finishes, the Pico should reboot automatically (typical UF2 behavior).

---

### Serial Monitoring

1. In **Administrator PowerShell**, attach the Pico to WSL:

```powershell
usbipd list
usbipd bind --busid <BUSID>          # admin once
usbipd attach --wsl --busid <BUSID>  # run each session
```

2. In WSL, find the serial device and start monitoring:

```bash
ls -l /dev/ttyACM* 2>/dev/null
tinygo monitor -port /dev/ttyACM0
```

If you see multiple `ttyACM` devices, pick the one that appears when you attach the Pico.
