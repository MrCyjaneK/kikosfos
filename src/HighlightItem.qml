// SPDX-FileCopyrightText: 2013-2023 Jolla Ltd.
// SPDX-FileCopyrightText: 2024-2025 Jolla Mobile Ltd
//
// SPDX-License-Identifier: BSD-3-Clause

import QtQuick 2.0
import Sailfish.Silica 1.0

/*!
  \inqmlmodule Sailfish.Gallery
*/
Rectangle {
    property bool active
    property real highlightOpacity: Theme.opacityHigh

    color: Theme.highlightBackgroundColor
    opacity: active ? highlightOpacity : 0.0

    Behavior on opacity {
        FadeAnimation {
            duration: 100
        }
    }
}
