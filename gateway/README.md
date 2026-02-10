To properly configure the gateway, you need to install bluez and enable bluetooth:
```bash
sudo apt update
sudo apt install -y bluez
sudo systemctl enable --now bluetooth
```