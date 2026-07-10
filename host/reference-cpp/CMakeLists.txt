cmake_minimum_required(VERSION 4.1)

project(velox_reference_cpp LANGUAGES NONE)

set(VELOX_WINDOWS_SDK_VERSION "10.0.26100.0" CACHE STRING "Windows SDK version")
set(VELOX_WINDOWS_KITS "C:/Program Files (x86)/Windows Kits/10")
get_filename_component(VELOX_ROOT "${CMAKE_CURRENT_SOURCE_DIR}/../.." ABSOLUTE)

set(WEBVIEW2_ROOT "${VELOX_ROOT}/.cache/deps/webview2/package/build/native")
set(WEBVIEW2_HEADER "${WEBVIEW2_ROOT}/include/WebView2.h")
set(WEBVIEW2_LOADER "${WEBVIEW2_ROOT}/x64/WebView2Loader.dll")
set(SDK_INCLUDE "${VELOX_WINDOWS_KITS}/Include/${VELOX_WINDOWS_SDK_VERSION}")
set(SDK_LIB "${VELOX_WINDOWS_KITS}/Lib/${VELOX_WINDOWS_SDK_VERSION}")
set(CLANG_RUNTIME "${VELOX_ROOT}/.pixi/envs/default/Library/lib/clang/21/lib/windows")

file(GLOB VC_TOOLSET_CANDIDATES
  "C:/Program Files/Microsoft Visual Studio/*/*/VC/Tools/MSVC/*")
list(SORT VC_TOOLSET_CANDIDATES COMPARE NATURAL ORDER DESCENDING)
set(VC_TOOLSET "")
foreach(candidate IN LISTS VC_TOOLSET_CANDIDATES)
  if(EXISTS "${candidate}/include/vcruntime.h")
    set(VC_TOOLSET "${candidate}")
    break()
  endif()
endforeach()
if(VC_TOOLSET STREQUAL "")
  message(FATAL_ERROR "A Visual Studio C++ header toolset is required by the Windows SDK headers")
endif()

find_program(CLANG_CL clang-cl REQUIRED)
find_program(LLD_LINK lld-link REQUIRED)

foreach(required_path
    "${WEBVIEW2_HEADER}"
    "${WEBVIEW2_LOADER}"
    "${SDK_INCLUDE}/um/windows.h"
    "${SDK_LIB}/um/x64/kernel32.lib"
    "${SDK_LIB}/ucrt/x64/ucrt.lib")
  if(NOT EXISTS "${required_path}")
    message(FATAL_ERROR "Required reference-host dependency is missing: ${required_path}")
  endif()
endforeach()

set(REFERENCE_OBJECT "${CMAKE_CURRENT_BINARY_DIR}/reference_host.obj")
set(REFERENCE_EXE "${CMAKE_CURRENT_BINARY_DIR}/velox-host-cpp.exe")

add_custom_command(
  OUTPUT "${REFERENCE_OBJECT}"
  COMMAND "${CLANG_CL}"
    /nologo
    /c
    /std:c++latest
    /W4
    /WX
    /Zl
    /GS-
    /GR-
    /EHs-c-
    /DUNICODE
    /D_UNICODE
    /DWIN32_LEAN_AND_MEAN
    /I${WEBVIEW2_ROOT}/include
    /imsvc${SDK_INCLUDE}/ucrt
    /imsvc${SDK_INCLUDE}/shared
    /imsvc${SDK_INCLUDE}/um
    /imsvc${SDK_INCLUDE}/winrt
    /imsvc${VC_TOOLSET}/include
    "${CMAKE_CURRENT_SOURCE_DIR}/main.cpp"
    /Fo${REFERENCE_OBJECT}
  DEPENDS "${CMAKE_CURRENT_SOURCE_DIR}/main.cpp" "${WEBVIEW2_HEADER}"
  VERBATIM
)

add_custom_command(
  OUTPUT "${REFERENCE_EXE}"
  COMMAND "${LLD_LINK}"
    /nologo
    /subsystem:windows
    /entry:wWinMainCRTStartup
    /nodefaultlib
    /opt:ref
    /opt:icf
    /libpath:${SDK_LIB}/um/x64
    /libpath:${SDK_LIB}/ucrt/x64
    /libpath:${CLANG_RUNTIME}
    "${REFERENCE_OBJECT}"
    kernel32.lib
    user32.lib
    ole32.lib
    shell32.lib
    advapi32.lib
    uuid.lib
    clang_rt.builtins-x86_64.lib
    /out:${REFERENCE_EXE}
  COMMAND "${CMAKE_COMMAND}" -E copy_if_different
    "${WEBVIEW2_LOADER}"
    "${CMAKE_CURRENT_BINARY_DIR}/WebView2Loader.dll"
  DEPENDS "${REFERENCE_OBJECT}" "${WEBVIEW2_LOADER}"
  VERBATIM
)

add_custom_target(velox_reference_cpp ALL DEPENDS "${REFERENCE_EXE}")
