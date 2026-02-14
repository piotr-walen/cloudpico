```
export PICO_SDK_PATH=~/pico/pico-sdk
```

```
rm -rf ./build && mkdir build 
cmake -S . -B build -G Ninja -DPICO_BOARD=pico2_w -DDEVICE_ID=0x12345678 -DPOLl_INTERVAL_MS=10000
cmake --build build -j
```


```
sudo ~/pico/picotool/build/picotool load ./build/cloudpico.uf2
```

```
screen /dev/ttyACM0 115200
```