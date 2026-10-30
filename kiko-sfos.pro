TARGET = kiko-sfos

CONFIG += sailfishapp

SOURCES += \
    src/main.cpp \
    src/kikoapi.cpp

HEADERS += \
    src/kikoapi.h \
    src/kiko_abi.h

# libkiko.so is installed on the default library path. Do not rpath the build tree.
QMAKE_LFLAGS_RPATH =

LIBS += -L$$PWD/go -lkiko

kikolib.files = $$PWD/go/libkiko.so
kikolib.path = $$[QT_INSTALL_LIBS]
INSTALLS += kikolib
