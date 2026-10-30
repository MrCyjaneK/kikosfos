# SPDX-FileCopyrightText: 2013-2021 Jolla Ltd.
# SPDX-FileCopyrightText: 2025 Jolla Mobile Ltd
#
# SPDX-License-Identifier: BSD-3-Clause

TEMPLATE = subdirs
SUBDIRS = tst_imageeditor

OTHER_FILES += auto/*.qml auto/*.js

check.commands += cd auto && qmltestrunner
QMAKE_EXTRA_TARGETS += check
