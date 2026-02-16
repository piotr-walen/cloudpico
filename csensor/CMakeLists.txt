cmake_minimum_required(VERSION 3.13)

# Export compile_commands.json for VSCode/Cursor IntelliSense (Pico SDK headers)
set(CMAKE_EXPORT_COMPILE_COMMANDS ON)

# include pico_sdk_import.cmake from the Pico SDK
include(pico_sdk_import.cmake)

project(cloudpico)

# Initialize the SDK
pico_sdk_init()

add_executable(cloudpico
    main.c
    flash_counter.c
    lib/bosch_bme280_api/bme280.c
    bme280_pico_i2c.c
    ble_advertise.c
)

# Add Bosch API submodule to include directories
target_include_directories(cloudpico PRIVATE lib/bosch_bme280_api)

# Add current directory for btstack_config.h and flash_counter.h
target_include_directories(cloudpico PRIVATE ${CMAKE_CURRENT_LIST_DIR})

# Link required libraries
target_link_libraries(cloudpico
    pico_stdlib
    pico_binary_info
    hardware_i2c
    hardware_flash
    hardware_sync
)

if (PICO_CYW43_SUPPORTED)
    # BLE support requires btstack libraries
    target_link_libraries(cloudpico
        pico_btstack_ble
        pico_btstack_cyw43
        pico_cyw43_arch_none
    )
endif()

# Enable USB serial output (stdio over USB) - must be after executable creation
pico_enable_stdio_usb(cloudpico 1)
pico_enable_stdio_uart(cloudpico 0)

set(DEVICE_ID "0x00000000" CACHE STRING "BLE device ID")
target_compile_definitions(cloudpico PRIVATE DEVICE_ID=${DEVICE_ID})

set(POLL_INTERVAL_MS "10000" CACHE STRING "Sensor poll interval in milliseconds")
target_compile_definitions(cloudpico PRIVATE POLL_INTERVAL_MS=${POLL_INTERVAL_MS})

set(ADVERTISEMENT_INTERVAL_MS "5000" CACHE STRING "BLE advertisement interval in milliseconds")
target_compile_definitions(cloudpico PRIVATE ADVERTISEMENT_INTERVAL_MS=${ADVERTISEMENT_INTERVAL_MS})

# create map/bin/hex file etc.
pico_add_extra_outputs(cloudpico)
