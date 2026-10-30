import QtQuick 2.6
import Sailfish.Silica 1.0

Column {
    id: root
    property bool immediate
    property string day
    property string clock
    property bool highlighted

    width: Math.max(dayMeasure.implicitWidth, clockMeasure.implicitWidth)
    spacing: 0

    Label {
        id: dayMeasure
        visible: false
        text: "Today"
        font.pixelSize: Theme.fontSizeExtraSmall
    }

    Label {
        id: clockMeasure
        visible: false
        text: "00:00"
        font.pixelSize: Theme.fontSizeSmall
    }

    Label {
        width: parent.width
        text: root.immediate ? "Now" : (root.day !== "" ? root.day : "Today")
        font.pixelSize: Theme.fontSizeExtraSmall
        color: root.immediate
               ? Theme.highlightColor
               : (root.highlighted ? Theme.secondaryHighlightColor : Theme.secondaryColor)
        truncationMode: TruncationMode.Fade
    }

    Label {
        width: parent.width
        text: root.clock
        font.pixelSize: Theme.fontSizeSmall
        color: root.highlighted ? Theme.highlightColor : Theme.primaryColor
        truncationMode: TruncationMode.Fade
    }
}
