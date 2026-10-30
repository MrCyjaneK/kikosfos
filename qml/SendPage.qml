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

            PageHeader { title: "Send" }

            TextField {
                id: recipient
                width: parent.width
                placeholderText: "Address"
            }

            TextField {
                id: amount
                width: parent.width
                placeholderText: "Amount"
                inputMethodHints: Qt.ImhFormattedNumbersOnly
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
                text: "Send"
                onClicked: {
                    if (kiko.walletSend(wallet, recipient.text, amount.text) !== 0)
                        errorText = kiko.lastError()
                }
            }
        }
    }
}
