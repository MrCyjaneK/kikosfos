TARGET = kiko-sfos

CONFIG += sailfishapp
QT += concurrent

INCLUDEPATH += $$PWD/go

SOURCES += \
    src/main.cpp \
    src/kikoapi.cpp \
    src/kikoimage.cpp

HEADERS += \
    src/kikoapi.h \
    src/kikoimage.h

# libkiko.so and libkiko.h come from `go build -buildmode=c-shared` before qmake.
# libkiko.so is installed on the default library path. Do not rpath the build tree.
QMAKE_LFLAGS_RPATH =

LIBS += -L$$PWD/go -lkiko

kikolib.files = $$PWD/go/libkiko.so
kikolib.path = $$[QT_INSTALL_LIBS]
INSTALLS += kikolib
