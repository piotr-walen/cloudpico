```
export PICO_SDK_PATH=~/pico/pico-sdk
```

```
rm -rf ./build && mkdir build 
cmake -S . -B build -G Ninja -DPICO_BOARD=pico2_w
cmake --build build -j
```


```
sudo ~/pico/picotool/build/picotool load ./build/cloudpico.uf2
```

```
screen /dev/ttyACM0 115200
```