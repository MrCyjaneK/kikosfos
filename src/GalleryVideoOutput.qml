// SPDX-FileCopyrightText: 2020-2023 Jolla Ltd.
// SPDX-FileCopyrightText: 2024-2025 Jolla Mobile Ltd
//
// SPDX-License-Identifier: BSD-3-Clause

import QtQuick 2.6
import Sailfish.Silica 1.0
import QtMultimedia 5.0

/*!
  \inqmlmodule Sailfish.Gallery
*/
VideoOutput {
    id: output

    property GalleryMediaPlayer player

    source: player
    visible: player && player.playbackState !== MediaPlayer.StoppedState

    BusyIndicator {
        z: 1
        anchors.centerIn: parent
        size: BusyIndicatorSize.Large
        opacity: 1.0
        visible: running
        parent: output.parent
        running: player && player.busy
    }
}
