# SPDX-FileCopyrightText: 2022 Jolla Ltd.
# SPDX-FileCopyrightText: 2025 Jolla Mobile Ltd
#
# SPDX-License-Identifier: BSD-3-Clause

TEMPLATE = aux

CONFIG += sailfish-qdoc-template
SAILFISH_QDOC.project = sailfish-gallery
SAILFISH_QDOC.config = sailfish-gallery.qdocconf
SAILFISH_QDOC.style = offline
SAILFISH_QDOC.path = /usr/share/doc/Sailfish/Gallery/

OTHER_FILES += $$PWD/sailfish-gallery.qdocconf \
               $$PWD/sailfish-gallery-qmltypes.qdoc
