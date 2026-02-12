// TTS.cpp C API for Go bindings
// This header exposes a C-compatible interface for the TTS.cpp library

#ifndef TTS_C_API_H
#define TTS_C_API_H

#include <stddef.h>
#include <stdbool.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

// Opaque handle to TTS runner
typedef struct tts_runner_handle tts_runner_handle;

// Response structure for audio data
typedef struct {
    float* data;
    size_t n_outputs;
    uint32_t sample_rate;
} tts_audio_data;

// Configuration structure
typedef struct {
    const char* voice;
    int top_k;
    float temperature;
    float repetition_penalty;
    float top_p;
    int max_tokens;
    bool use_cross_attn;
    const char* espeak_voice_id;
} tts_config;

// Initialize configuration with defaults
tts_config tts_config_default();

// Create TTS runner from model file
// Returns NULL on failure
tts_runner_handle* tts_runner_create(
    const char* model_path,
    int n_threads,
    tts_config* config,
    bool cpu_only
);

// Generate audio from text
// Returns true on success, false on failure
bool tts_generate(
    tts_runner_handle* runner,
    const char* text,
    tts_audio_data* output
);

// Free audio data memory
void tts_audio_data_free(tts_audio_data* data);

// Free TTS runner
void tts_runner_free(tts_runner_handle* runner);

// Get error message from last operation
const char* tts_get_error();

// Get list of available voices
// Returns array of voice names (NULL terminated)
char** tts_list_voices(tts_runner_handle* runner);

// Free voices list
void tts_free_voices(char** voices);

// Update conditional prompt (for models that support it)
bool tts_update_conditional_prompt(
    tts_runner_handle* runner,
    const char* text_encoder_path,
    const char* prompt
);

// Check if model supports voice selection
bool tts_supports_voices(tts_runner_handle* runner);

// Get supported architecture names
char** tts_get_supported_architectures();

// Free architecture list
void tts_free_architectures(char** archs);

// Save audio data to WAV file
bool tts_save_audio_wav(
    tts_audio_data* data,
    const char* file_path,
    float sample_rate
);

// Get audio data as bytes (16-bit PCM)
// Returns pointer to buffer (must be freed by caller)
unsigned char* tts_audio_to_bytes(
    tts_audio_data* data,
    size_t* out_size,
    int* out_channels
);

#ifdef __cplusplus
}
#endif

#endif // TTS_C_API_H
