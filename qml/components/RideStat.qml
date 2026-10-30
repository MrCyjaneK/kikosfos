import QtQuick 2.6
import Sailfish.Silica 1.0

Column {
    id: root
    property string value
    property string caption
    property bool highlighted

    spacing: 0

    Row {
        spacing: Theme.paddingSmall

        Label {
            id: valueLabel
            text: root.value
            font.pixelSize: Theme.fontSizeMedium
            color: root.highlighted ? Theme.highlightColor : Theme.primaryColor
        }

        Label {
            anchors.baseline: valueLabel.baseline
            text: "km"
            font.pixelSize: Theme.fontSizeSmall
            color: root.highlighted ? Theme.secondaryHighlightColor : Theme.secondaryColor
        }
    }

    Label {
        text: root.caption
        font.pixelSize: Theme.fontSizeSmall
        color: root.highlighted ? Theme.secondaryHighlightColor : Theme.secondaryColor
    }
}
