import QtQuick 2.6
import Sailfish.Silica 1.0

Page {
    id: page
    property int wallet
    property var coinPage
    property string mnemonic
    property string privateKey
    property string errorText

    onStatusChanged: {
        if (status === PageStatus.Active)
            load()
    }

    function load() {
        errorText = ""
        mnemonic = kiko.walletMnemonic(wallet)
        if (mnemonic === "")
            errorText = kiko.lastError()
        privateKey = kiko.walletPrivateKey(wallet)
        if (privateKey === "" && errorText === "")
            errorText = kiko.lastError()
    }

    SilicaFlickable {
        anchors.fill: parent
        contentHeight: column.height

        Column {
            id: column
            width: parent.width
            spacing: Theme.paddingLarge

            PageHeader { title: "Settings" }

            Label {
                visible: page.errorText !== ""
                width: parent.width - 2 * Theme.horizontalPageMargin
                x: Theme.horizontalPageMargin
                wrapMode: Text.Wrap
                color: Theme.highlightColor
                text: page.errorText
            }

            TextArea {
                width: parent.width
                readOnly: true
                label: "Seed phrase"
                text: page.mnemonic
                wrapMode: TextEdit.Wrap
            }

            TextArea {
                width: parent.width
                readOnly: true
                label: "Private key"
                text: page.privateKey
                wrapMode: TextEdit.Wrap
            }

            Button {
                anchors.horizontalCenter: parent.horizontalCenter
                text: "Delete wallet"
                onClicked: pageStack.push(Qt.resolvedUrl("ConfirmPinPage.qml"), {
                    wallet: page.wallet,
                    coinPage: page.coinPage
                })
            }
        }
    }
}
