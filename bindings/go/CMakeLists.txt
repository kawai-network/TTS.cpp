# C API Library for Go bindings
# This library wraps the C++ TTS library in a C-compatible interface

# C API Library
add_library(tts_c_api SHARED
    internal/tts_c_api.cpp
    internal/tts_c_api.h
)

# Include directories
target_include_directories(tts_c_api PUBLIC
    ${CMAKE_CURRENT_SOURCE_DIR}/internal
    ${CMAKE_CURRENT_SOURCE_DIR}/../../include
    ${CMAKE_CURRENT_SOURCE_DIR}/../../src
    ${CMAKE_CURRENT_SOURCE_DIR}/../../ggml/include
    ${CMAKE_CURRENT_SOURCE_DIR}/../../ggml/src
)

# Link against the main TTS library
target_link_libraries(tts_c_api PUBLIC tts ggml)

# Set properties for shared library
set_target_properties(tts_c_api PROPERTIES
    POSITION_INDEPENDENT_CODE ON
    C_VISIBILITY_PRESET hidden
    CXX_VISIBILITY_PRESET hidden
    VISIBILITY_INLINES_HIDDEN YES
)

# Export all symbols (needed for C API)
if (WIN32)
    set_target_properties(tts_c_api PROPERTIES
        WINDOWS_EXPORT_ALL_SYMBOLS ON
    )
endif()

# Compiler definitions
if (BUILD_SHARED_LIBS)
    target_compile_definitions(tts_c_api PRIVATE TTS_BUILD_SHARED)
    target_compile_definitions(tts_c_api PUBLIC TTS_SHARED)
endif()

# Installation
install(TARGETS tts_c_api
    LIBRARY DESTINATION ${CMAKE_INSTALL_LIBDIR}
    RUNTIME DESTINATION ${CMAKE_INSTALL_BINDIR}
    ARCHIVE DESTINATION ${CMAKE_INSTALL_LIBDIR}
)

install(FILES internal/tts_c_api.h
    DESTINATION ${CMAKE_INSTALL_INCLUDEDIR}/tts
)
