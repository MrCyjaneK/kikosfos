import QtQuick 2.6
import Sailfish.Silica 1.0

Item {
    id: root
    property string caption
    property string amount
    property string euro
    property bool highlighted
    property int captionWidth

    width: parent.width
    height: Math.max(captionLabel.implicitHeight, amountLabel.implicitHeight, euroLabel.implicitHeight)

    Label {
        id: captionLabel
        anchors.left: parent.left
        anchors.verticalCenter: parent.verticalCenter
        width: root.captionWidth > 0 ? root.captionWidth : implicitWidth
        text: root.caption
        font.pixelSize: Theme.fontSizeSmall
        color: root.highlighted ? Theme.secondaryHighlightColor : Theme.secondaryColor
        verticalAlignment: Text.AlignVCenter
    }

    Label {
        id: euroLabel
        anchors.right: parent.right
        anchors.verticalCenter: parent.verticalCenter
        visible: root.euro !== ""
        text: root.euro
        font.pixelSize: Theme.fontSizeSmall
        color: root.highlighted ? Theme.secondaryHighlightColor : Theme.secondaryColor
        verticalAlignment: Text.AlignVCenter
    }

    Label {
        id: amountLabel
        anchors.left: captionLabel.right
        anchors.leftMargin: Theme.paddingSmall
        anchors.right: euroLabel.visible ? euroLabel.left : parent.right
        anchors.rightMargin: euroLabel.visible ? Theme.paddingMedium : 0
        anchors.verticalCenter: parent.verticalCenter
        text: root.amount
        font.pixelSize: Theme.fontSizeSmall
        color: root.highlighted ? Theme.highlightColor : Theme.primaryColor
        truncationMode: TruncationMode.Fade
        verticalAlignment: Text.AlignVCenter
    }
}
