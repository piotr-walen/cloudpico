# BME280 Pin Connections for Raspberry Pi Pico

## Standard Connections (Default I2C Pins)

| Raspberry Pi Pico | BME280 Sensor |
|-------------------|---------------|
| **Pin 6 (GP4)**   | SDA           |
| **Pin 7 (GP5)**   | SCL           |
| **Pin 36 (3V3)**  | VCC/VIN       |
| **Pin 38 (GND)**  | GND           |

## LED Connection

| Raspberry Pi Pico | LED Diode |
|-------------------|-----------|
| **Pin 1 (GP0)**   | Anode (+) |
| **Pin 38 (GND)**  | Cathode (-) |

### Pin Details:
- **GP4 (Pin 6)**: I2C Data Line (SDA) - `PICO_DEFAULT_I2C_SDA_PIN`
- **GP5 (Pin 7)**: I2C Clock Line (SCL) - `PICO_DEFAULT_I2C_SCL_PIN`
- **3V3 (Pin 36)**: 3.3V Power Supply
- **GND (Pin 38)**: Ground
- **GP0 (Pin 1)**: External LED Control (Anode) - Toggles to show sensor activity

