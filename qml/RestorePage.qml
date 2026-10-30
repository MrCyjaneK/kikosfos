import QtQuick 2.6
import Sailfish.Silica 1.0

Page {
    property int wallet
    property string errorText

    SilicaFlickable {
        anchors.fill: parent
        contentHeight: column.height

        Column {
            id: column
            width: parent.width
            spacing: Theme.paddingLarge

            PageHeader { title: "Restore" }

            TextArea {
                id: phrase
                width: parent.width
                placeholderText: "Recovery phrase"
                wrapMode: TextEdit.Wrap
            }

            Label {
                visible: errorText !== ""
                width: parent.width - 2 * Theme.horizontalPageMargin
                x: Theme.horizontalPageMargin
                wrapMode: Text.Wrap
                color: Theme.highlightColor
                text: errorText
            }

            Button {
                anchors.horizontalCenter: parent.horizontalCenter
                text: "Restore"
                onClicked: {
                    if (kiko.walletRestore(wallet, phrase.text) === 0)
                        pageStack.pop()
                    else
                        errorText = kiko.lastError()
                }
            }
        }
    }
}
