```
export PICO_SDK_PATH=~/pico/pico-sdk
```

```
rm -rf ./build && mkdir build 
cmake -S . -B build -G Ninja -DPICO_BOARD=pico2_w -DDEVICE_ID=0x0001 -DPOLL_INTERVAL_MS=60000 -DADVERTISEMENT_INTERVAL_MS=1000
cmake --build build -j
```

Note: `DEVICE_ID` is stored as uint32_t but only the lower 16 bits (0xFFFF) are transmitted in BLE advertisements. Defaults: `POLL_INTERVAL_MS=10000`, `ADVERTISEMENT_INTERVAL_MS=5000`.


```
sudo ~/pico/picotool/build/picotool load ./build/cloudpico.uf2
```

```
screen /dev/ttyACM0 115200
```