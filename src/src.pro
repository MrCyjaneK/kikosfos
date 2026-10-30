# SPDX-FileCopyrightText: 2013-2023 Jolla Ltd.
# SPDX-FileCopyrightText: 2024-2025 Jolla Mobile Ltd
#
# SPDX-License-Identifier: BSD-3-Clause

TEMPLATE = lib
TARGET = sailfishgalleryplugin
TARGET = $$qtLibraryTarget($$TARGET)

MODULENAME = Sailfish/Gallery
TARGETPATH = $$[QT_INSTALL_QML]/$$MODULENAME

QT += qml quick concurrent
CONFIG += plugin link_pkgconfig hide_symbols

contains(CONFIG, desktop) {
    DEFINES *= DESKTOP
} else {
    packagesExist(quillmetadata-qt5) {
        PKGCONFIG += quillmetadata-qt5
    }
}

import.files = *.qml qmldir plugins.qmltypes scripts
import.path = $$TARGETPATH

private.files = private/*.qml private/qmldir private/plugins.qmltypes
private.path = $$TARGETPATH/private

privatesymlink.CONFIG += no_check_exist
privatesymlink.files = private/libsailfishgalleryplugin.so
privatesymlink.path = $$TARGETPATH/private

target.path = $$TARGETPATH

OTHER_FILES += \
    *.qml \
    private/*.qml \
    private/libsailfishgalleryplugin.so

SOURCES += \
    declarativeimageeditor.cpp \
    declarativeimageeditor_p.cpp \
    plugin.cpp \
    declarativeimagemetadata.cpp \
    declarativeavatarfilehandler.cpp

HEADERS += declarativeimageeditor.h \
    declarativeimageeditor_p.h \
    declarativeimagemetadata.h \
    declarativeavatarfilehandler.h

TS_FILE = $$OUT_PWD/sailfish_components_gallery_qt5.ts
EE_QM = $$OUT_PWD/sailfish_components_gallery_qt5_eng_en.qm

translations.commands += lupdate $$PWD -ts $$TS_FILE
translations.depends = $$PWD/*.qml
translations.CONFIG += no_check_exist no_link
translations.output = $$TS_FILE
translations.input = .

translations_install.files = $$TS_FILE
translations_install.path = /usr/share/translations/source
translations_install.CONFIG += no_check_exist

# should add -markuntranslated "-" when proper translations are in place (or for testing)
engineering_english.commands += lrelease -idbased $$TS_FILE -qm $$EE_QM
engineering_english.CONFIG += no_check_exist no_link
engineering_english.depends = translations
engineering_english.input = $$TS_FILE
engineering_english.output = $$EE_QM

engineering_english_install.path = /usr/share/translations
engineering_english_install.files = $$EE_QM
engineering_english_install.CONFIG += no_check_exist

QMAKE_EXTRA_TARGETS += translations engineering_english

PRE_TARGETDEPS += translations engineering_english

INSTALLS += import private privatesymlink target translations_install engineering_english_install

# HACK: pass -nocomposites to work around the issue with types leaked
# (mostly) from Silica. All of these are composite types and we have no other
# composite types.
qmltypes.commands = qmlplugindump -noinstantiate -nonrelocatable -nocomposites \
         Sailfish.Gallery 1.0 > $$PWD/plugins.qmltypes
QMAKE_EXTRA_TARGETS += qmltypes
