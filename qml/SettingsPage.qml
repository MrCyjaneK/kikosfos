import QtQuick 2.6
import Sailfish.Silica 1.0
import "components"

Page {
    id: page
    property int wallet
    property var coinPage
    property string mnemonic
    property string privateKey
    property string multisigMaterial
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
        multisigMaterial = kiko.walletMultisigMaterial(wallet)
        if (multisigMaterial === "" && errorText === "")
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

            SecretField {
                label: "Seed phrase"
                value: page.mnemonic
            }

            SecretField {
                label: "Private key"
                value: page.privateKey
            }

            SecretField {
                label: "Multisig material"
                value: page.multisigMaterial
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
