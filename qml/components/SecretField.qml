import QtQuick 2.6
import Sailfish.Silica 1.0

ListItem {
    id: root
    property string label
    property string value

    width: parent.width
    contentHeight: field.height

    menu: ContextMenu {
        MenuItem {
            text: "Copy"
            onClicked: Clipboard.text = root.value
        }
    }

    TextArea {
        id: field
        width: parent.width
        readOnly: true
        label: root.label
        text: root.value
        wrapMode: TextEdit.Wrap
    }

    MouseArea {
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.top: parent.top
        height: field.height
        onPressAndHold: root.openMenu()
    }
}
