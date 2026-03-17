#include "flash_counter.h"

#include <string.h>
#include "pico/stdlib.h"
#include "pico/flash.h"
#include "hardware/flash.h"
#include "hardware/sync.h"

/*
    Configuration

    By default we use the LAST flash sector (4KB).
    If you ever want to move this storage area, you can change these macros.

    WARNING:
    - Do not place this region where your program binary or filesystem writes.
*/

#ifndef FLASH_COUNTER_SECTOR_OFFSET
#define FLASH_COUNTER_SECTOR_OFFSET (PICO_FLASH_SIZE_BYTES - FLASH_SECTOR_SIZE)
#endif

#define FLASH_COUNTER_XIP_BASE      (XIP_BASE + FLASH_COUNTER_SECTOR_OFFSET)

// “Magic” + version to identify our record format
#define FLASH_COUNTER_MAGIC         0x524E434Eu  // 'RNCN' arbitrary tag
#define FLASH_COUNTER_VERSION       1u

// Record stored in one flash page (256 bytes).
typedef struct __attribute__((packed)) {
    uint32_t magic;
    uint32_t version;
    uint32_t seq;       // increasing sequence number
    uint32_t counter;   // your stored counter
    uint32_t crc;       // CRC32 of entire record with crc field set to 0
    uint8_t  pad[FLASH_PAGE_SIZE - 5u * 4u];
} flash_counter_record_t;

_Static_assert(sizeof(flash_counter_record_t) == FLASH_PAGE_SIZE, "Record must be exactly one flash page");

/* ---------------- CRC32 (simple, good enough) ---------------- */

static uint32_t crc32_simple(const void *data, size_t len) {
    uint32_t crc = 0xFFFFFFFFu;
    const uint8_t *p = (const uint8_t *)data;

    while (len--) {
        crc ^= *p++;
        for (int i = 0; i < 8; i++) {
            crc = (crc >> 1) ^ (0xEDB88320u & (-(int)(crc & 1)));
        }
    }
    return ~crc;
}

static bool record_valid(const flash_counter_record_t *r) {
    if (r->magic != FLASH_COUNTER_MAGIC) return false;
    if (r->version != FLASH_COUNTER_VERSION) return false;

    flash_counter_record_t tmp;
    memcpy(&tmp, r, sizeof(tmp));
    uint32_t stored = tmp.crc;
    tmp.crc = 0;

    uint32_t computed = crc32_simple(&tmp, sizeof(tmp));
    return computed == stored;
}

/* ---------------- Low-level flash ops via flash_safe_execute ---------------- */

// Erase one sector (4KB). Must be run via flash_safe_execute().
static void do_erase_sector(void *param) {
    (void)param;
    flash_range_erase(FLASH_COUNTER_SECTOR_OFFSET, FLASH_SECTOR_SIZE);
}

// Program one page (256B). Must be run via flash_safe_execute().
// param points to uintptr_t params[2] = { offset, data_ptr }.
static void do_program_page(void *param) {
    uintptr_t *p = (uintptr_t *)param;
    uint32_t offset = (uint32_t)p[0];
    const uint8_t *data = (const uint8_t *)p[1];
    flash_range_program(offset, data, FLASH_PAGE_SIZE);
}

static void erase_storage_sector(void) {
    int rc = flash_safe_execute(do_erase_sector, NULL, UINT32_MAX);
    hard_assert(rc == PICO_OK);
}

static void program_storage_page(const flash_counter_record_t *rec, uint32_t page_index) {
    uint32_t page_offs = FLASH_COUNTER_SECTOR_OFFSET + page_index * FLASH_PAGE_SIZE;

    uintptr_t params[2] = {
        (uintptr_t)page_offs,
        (uintptr_t)rec
    };

    int rc = flash_safe_execute(do_program_page, params, UINT32_MAX);
    hard_assert(rc == PICO_OK);
}

/* ---------------- Storage scanning helpers ---------------- */

static const flash_counter_record_t *storage_pages(void) {
    return (const flash_counter_record_t *)FLASH_COUNTER_XIP_BASE;
}

// Find:
// - latest valid record (highest seq)
// - first free page (magic == 0xFFFFFFFF)
static void scan_storage(int *out_latest_index,
                         uint32_t *out_latest_seq,
                         int *out_first_free_index) {
    const flash_counter_record_t *pages = storage_pages();

    int latest = -1;
    uint32_t latest_seq = 0;
    int first_free = -1;

    const int num_pages = (FLASH_SECTOR_SIZE / FLASH_PAGE_SIZE);

    for (int i = 0; i < num_pages; i++) {
        const flash_counter_record_t *r = &pages[i];

        // Unused flash reads as 0xFF... so magic is 0xFFFFFFFF
        if (r->magic == 0xFFFFFFFFu) {
            first_free = i;
            break;
        }

        if (record_valid(r)) {
            if (latest == -1 || r->seq > latest_seq) {
                latest = i;
                latest_seq = r->seq;
            }
        }
    }

    *out_latest_index = latest;
    *out_latest_seq = latest_seq;
    *out_first_free_index = first_free;
}

/* ---------------- Public API ---------------- */

bool flash_counter_read(uint32_t *out_counter) {
    if (!out_counter) return false;

    int latest;
    uint32_t latest_seq;
    int first_free;
    scan_storage(&latest, &latest_seq, &first_free);

    if (latest < 0) {
        *out_counter = 0;
        return false;
    }

    const flash_counter_record_t *pages = storage_pages();
    *out_counter = pages[latest].counter;
    return true;
}

static bool write_new_record(uint32_t new_counter, uint32_t *out_value_written) {
    int latest;
    uint32_t latest_seq;
    int first_free;
    scan_storage(&latest, &latest_seq, &first_free);

    // If full, erase and start from page 0 again
    if (first_free < 0) {
        erase_storage_sector();
        first_free = 0;
        // After erase, latest_seq is no longer meaningful, but we can keep seq monotonic
        // by continuing from the previous latest_seq+1 (stored in RAM at boot time).
        // Here we just reuse latest_seq and increment it below.
    }

    uint32_t seq = latest_seq + 1;

    // Prepare record
    flash_counter_record_t rec;
    memset(&rec, 0xFF, sizeof(rec)); // typical flash-friendly init

    rec.magic = FLASH_COUNTER_MAGIC;
    rec.version = FLASH_COUNTER_VERSION;
    rec.seq = seq;
    rec.counter = new_counter;

    rec.crc = 0;
    rec.crc = crc32_simple(&rec, sizeof(rec));

    // Program to flash
    program_storage_page(&rec, (uint32_t)first_free);

    if (out_value_written) *out_value_written = new_counter;
    return true;
}

bool flash_counter_increment_and_store(uint32_t *out_counter) {
    uint32_t current = 0;
    (void)flash_counter_read(&current); // if none found, current stays 0

    uint32_t next = current + 1;
    return write_new_record(next, out_counter);
}

bool flash_counter_set(uint32_t value, uint32_t *out_value_written) {
    return write_new_record(value, out_value_written);
}
