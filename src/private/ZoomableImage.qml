// SPDX-FileCopyrightText: 2013-2023 Jolla Ltd.
// SPDX-FileCopyrightText: 2024-2025 Jolla Mobile Ltd
//
// SPDX-License-Identifier: BSD-3-Clause

import QtQuick 2.0
import QtGraphicalEffects 1.0
import Sailfish.Silica 1.0
import ".."

ImageViewer {
    id: root

    property int baseRotation
    property int imageRotation
    property alias brightness: adjustLevels.brightness
    property alias contrast: adjustLevels.contrast
    readonly property bool longPressed: pressed && !delayPressTimer.running
    property bool animatingBrightnessContrast

    contentRotation: baseRotation + imageRotation

    onAnimatingBrightnessContrastChanged: adjustLevels.visible = true

    function rotate(angle) {
        resetZoom()
        // Don't wait for the rotation animation to complete to new image dimensions
        transposeBinding.value = (baseRotation + rotationAnimation.to + angle) % 180
        rotationAnimation.to = rotationAnimation.to + angle
        rotationAnimation.restart()
    }

    Binding {
        id: transposeBinding

        target: root
        when: rotationAnimation.running
        property: "transpose"
    }

    NumberAnimation {
        id: rotationAnimation

        target: root
        property: "imageRotation"
        easing.type: Easing.InOutQuad
        duration: 200
    }

    Behavior on zoom {
        enabled: rotationAnimation.running
        SmoothedAnimation { duration: 200 }
    }

    BrightnessContrast {
        id: adjustLevels

        source: root
        visible: false
        cached: !animatingBrightnessContrast
        parent: root.parent
        width: source.width
        height: source.height
    }

    Timer {
        id: delayPressTimer

        running: pressed
        interval: 300
    }

    states: State {
        when: longPressed
        PropertyChanges {
            target: root
            brightness: 0.0
            contrast: 0.0
        }
    }
}
