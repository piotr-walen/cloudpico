cmake_minimum_required(VERSION 3.13)

# include pico_sdk_import.cmake from the Pico SDK
include(pico_sdk_import.cmake)

project(cloudpico)

# Initialize the SDK
pico_sdk_init()

add_executable(cloudpico
    main.c
    lib/bosch_bme280_api/bme280.c
    bme280_pico_i2c.c
    ble_advertise.c
)

# Add Bosch API submodule to include directories
target_include_directories(cloudpico PRIVATE lib/bosch_bme280_api)

# Add current directory for btstack_config.h
target_include_directories(cloudpico PRIVATE ${CMAKE_CURRENT_LIST_DIR})

# Link required libraries
target_link_libraries(cloudpico pico_stdlib pico_binary_info hardware_i2c)

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



# create map/bin/hex file etc.
pico_add_extra_outputs(cloudpico)