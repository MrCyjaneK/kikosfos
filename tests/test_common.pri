# SPDX-FileCopyrightText: 2013-2017 Jolla Ltd.
# SPDX-FileCopyrightText: 2025 Jolla Mobile Ltd
#
# SPDX-License-Identifier: BSD-3-Clause

TEMPLATE = app
QT += qml quick testlib concurrent

MODULENAME = Sailfish/Gallery
DEFINES *= MODULENAME=\"\\\"\"$${MODULENAME}\"\\\"\"

DEFINES += COMPONENTDIR=\\\"$$[QT_INSTALL_QML]/$$MODULENAME\\\"

contains(CONFIG, desktop) {
    DEFINES *= DESKTOP
    DEFINES += APPLICATIONDIR=\\\"$$PWD/../../applications/\\\"
} else {
    DEFINES += APPLICATIONDIR=\\\"/usr/share/\\\"

    # install the test
    target.path = /opt/tests/sailfish-components-gallery-qt5
    INSTALLS += target
}
