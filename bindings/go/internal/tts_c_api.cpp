// TTS.cpp C API implementation
#include "tts_c_api.h"
#include "../../include/common.h"
#include "../../src/models/loaders.h"
#include "../../include/audio_file.h"
#include <cstring>
#include <string>
#include <vector>
#include <memory>

static thread_local std::string g_last_error;

struct tts_runner_handle {
    std::unique_ptr<tts_generation_runner> runner;
    float sampling_rate;
};

tts_config tts_config_default() {
    tts_config config;
    config.voice = "";
    config.top_k = 50;
    config.temperature = 1.0f;
    config.repetition_penalty = 1.0f;
    config.top_p = 1.0f;
    config.max_tokens = 0;
    config.use_cross_attn = true;
    config.espeak_voice_id = "";
    return config;
}

tts_runner_handle* tts_runner_create(
    const char* model_path,
    int n_threads,
    tts_config* config,
    bool cpu_only
) {
    if (!model_path || !config) {
        g_last_error = "Invalid arguments";
        return nullptr;
    }

    try {
        generation_configuration gen_config(
            config->voice ? config->voice : "",
            config->top_k,
            config->temperature,
            config->repetition_penalty,
            config->use_cross_attn,
            config->espeak_voice_id ? config->espeak_voice_id : "",
            config->max_tokens,
            config->top_p
        );

        auto runner = runner_from_file(model_path, n_threads, gen_config, cpu_only);
        if (!runner) {
            g_last_error = "Failed to load model";
            return nullptr;
        }

        tts_runner_handle* handle = new tts_runner_handle();
        handle->runner = std::move(runner);
        handle->sampling_rate = handle->runner->sampling_rate;
        return handle;
    } catch (const std::exception& e) {
        g_last_error = e.what();
        return nullptr;
    }
}

bool tts_generate(
    tts_runner_handle* runner,
    const char* text,
    tts_audio_data* output
) {
    if (!runner || !text || !output) {
        g_last_error = "Invalid arguments";
        return false;
    }

    try {
        tts_response response;
        generation_configuration config = generation_configuration(); // Use defaults
        
        runner->runner->generate(text, response, config);
        
        if (response.n_outputs == 0) {
            g_last_error = "Empty response from model";
            return false;
        }

        // Copy data to output
        output->n_outputs = response.n_outputs;
        output->sample_rate = static_cast<uint32_t>(runner->sampling_rate);
        output->data = new float[response.n_outputs];
        std::memcpy(output->data, response.data, response.n_outputs * sizeof(float));
        
        return true;
    } catch (const std::exception& e) {
        g_last_error = e.what();
        return false;
    }
}

void tts_audio_data_free(tts_audio_data* data) {
    if (data && data->data) {
        delete[] data->data;
        data->data = nullptr;
        data->n_outputs = 0;
    }
}

void tts_runner_free(tts_runner_handle* runner) {
    delete runner;
}

const char* tts_get_error() {
    return g_last_error.c_str();
}

char** tts_list_voices(tts_runner_handle* runner) {
    if (!runner || !runner->runner) {
        return nullptr;
    }

    try {
        auto voices = runner->runner->list_voices();
        if (voices.empty()) {
            // Return empty list (just NULL terminator)
            char** result = new char*[1];
            result[0] = nullptr;
            return result;
        }

        char** result = new char*[voices.size() + 1];
        for (size_t i = 0; i < voices.size(); i++) {
            result[i] = new char[voices[i].size() + 1];
            std::strcpy(result[i], std::string(voices[i]).c_str());
        }
        result[voices.size()] = nullptr;
        return result;
    } catch (const std::exception& e) {
        g_last_error = e.what();
        return nullptr;
    }
}

void tts_free_voices(char** voices) {
    if (!voices) return;
    
    for (int i = 0; voices[i] != nullptr; i++) {
        delete[] voices[i];
    }
    delete[] voices;
}

bool tts_update_conditional_prompt(
    tts_runner_handle* runner,
    const char* text_encoder_path,
    const char* prompt
) {
    if (!runner || !text_encoder_path || !prompt) {
        g_last_error = "Invalid arguments";
        return false;
    }

    try {
        runner->runner->update_conditional_prompt(text_encoder_path, prompt);
        return true;
    } catch (const std::exception& e) {
        g_last_error = e.what();
        return false;
    }
}

bool tts_supports_voices(tts_runner_handle* runner) {
    if (!runner || !runner->runner) {
        return false;
    }
    return runner->runner->supports_voices;
}

char** tts_get_supported_architectures() {
    const char* archs[] = {"parler-tts", "kokoro", "dia", "orpheus", nullptr};
    int count = 4;
    
    char** result = new char*[count + 1];
    for (int i = 0; i < count; i++) {
        result[i] = new char[std::strlen(archs[i]) + 1];
        std::strcpy(result[i], archs[i]);
    }
    result[count] = nullptr;
    return result;
}

void tts_free_architectures(char** archs) {
    tts_free_voices(archs);  // Same implementation
}

bool tts_save_audio_wav(
    tts_audio_data* data,
    const char* file_path,
    float sample_rate
) {
    if (!data || !data->data || !file_path) {
        g_last_error = "Invalid arguments";
        return false;
    }

    try {
        AudioFile<float> audio_file;
        audio_file.setNumChannels(1);  // Mono
        audio_file.setNumSamplesPerChannel(static_cast<int>(data->n_outputs));
        audio_file.setSampleRate(static_cast<uint32_t>(sample_rate));
        audio_file.setBitDepth(16);

        // Copy samples
        for (size_t i = 0; i < data->n_outputs; i++) {
            audio_file.samples[0][i] = data->data[i];
        }

        return audio_file.save(file_path, AudioFileFormat::Wave);
    } catch (const std::exception& e) {
        g_last_error = e.what();
        return false;
    }
}

unsigned char* tts_audio_to_bytes(
    tts_audio_data* data,
    size_t* out_size,
    int* out_channels
) {
    if (!data || !data->data || !out_size || !out_channels) {
        g_last_error = "Invalid arguments";
        return nullptr;
    }

    try {
        *out_channels = 1;  // Mono
        *out_size = data->n_outputs * sizeof(float);
        
        unsigned char* bytes = new unsigned char[*out_size];
        std::memcpy(bytes, data->data, *out_size);
        return bytes;
    } catch (const std::exception& e) {
        g_last_error = e.what();
        return nullptr;
    }
}
