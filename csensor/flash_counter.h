#pragma once

#include <stdint.h>
#include <stdbool.h>

/*
    flash_counter module

    Stores a monotonically increasing counter in onboard QSPI flash.
    - Uses one 4KB sector (16 pages of 256 bytes)
    - Appends records until full, then erases sector and starts again
    - Uses CRC to reject partially written/corrupted records

    Typical use:
        uint32_t c;
        flash_counter_increment_and_store(&c);
*/

#ifdef __cplusplus
extern "C" {
#endif

// Reads the latest valid counter from flash (does NOT change it).
// Returns true if a valid record was found, false if none (counter is set to 0).
bool flash_counter_read(uint32_t *out_counter);

// Increments the counter by 1 and stores it to flash.
// Returns true on success. The new value is written to out_counter.
bool flash_counter_increment_and_store(uint32_t *out_counter);

// Optional: reset counter to 0 (writes a record with counter=0, seq bumped).
// This does not "erase everything" unless sector is full; it just stores a new record.
// Returns true on success.
bool flash_counter_set(uint32_t value, uint32_t *out_value_written);

#ifdef __cplusplus
}
#endif
