#include <stdint.h>
#include <stddef.h>

__attribute__((import_module("yotta_plugin_v1"), import_name("exchange")))
extern int64_t yotta_exchange(uint32_t request_ptr, uint32_t request_len,
                              uint32_t response_ptr, uint32_t response_cap);

static uint8_t arena[64 * 1024];
static uint32_t cursor;

__attribute__((export_name("yotta_alloc")))
uint32_t yotta_alloc(uint32_t size) {
  if (size > sizeof(arena) - cursor) return 0;
  uint32_t result = (uint32_t)(uintptr_t)&arena[cursor];
  cursor += size;
  return result;
}

// Canonical successful Result frame from the generated conformance vectors.
static const uint8_t success[] = {
  0x0a, 0x0e, 0x79, 0x6f, 0x74, 0x74, 0x61, 0x2e,
  0x70, 0x6c, 0x75, 0x67, 0x69, 0x6e, 0x2f, 0x31,
  0x10, 0x02, 0x92, 0x01, 0x0f, 0x08, 0x01, 0x2a,
  0x0b, 0x63, 0x6f, 0x6f, 0x70, 0x65, 0x72, 0x61,
  0x74, 0x69, 0x76, 0x65,
};

__attribute__((export_name("yotta_run")))
uint32_t yotta_run(uint32_t invocation_ptr, uint32_t invocation_len) {
  (void)invocation_ptr;
  (void)invocation_len;
  return yotta_exchange((uint32_t)(uintptr_t)success, sizeof(success), 0, 0) == 0 ? 0 : 1;
}

