# SPDX-FileCopyrightText: 2023 Jolla Ltd.
# SPDX-FileCopyrightText: 2025 Jolla Mobile Ltd
#
# SPDX-License-Identifier: BSD-3-Clause

# The only purpose of this subproject is invocation of qmlplugindump. All other
# stuff should go to the parent project.

TEMPLATE = aux

# HACK: pass -nocomposites to work around the issue with types leaked
# (mostly) from Silica. All of these are composite types and we have no other
# composite types.
qmltypes.commands = qmlplugindump -noinstantiate -nonrelocatable -nocomposites \
         Sailfish.Gallery.private 1.0 > $$PWD/plugins.qmltypes
QMAKE_EXTRA_TARGETS += qmltypes
