import QtQuick 2.6
import Sailfish.Silica 1.0

Page {
    id: page
    objectName: "coinPage"
    property int wallet
    property int exists: -1
    property string address
    property string errorText

    onStatusChanged: {
        if (status === PageStatus.Active)
            refresh()
    }

    function refresh() {
        errorText = ""
        exists = kiko.walletExists(wallet)
        if (exists === 1) {
            address = kiko.walletAddress(wallet)
            if (address === "") {
                errorText = kiko.lastError()
                exists = -1
            }
        } else {
            address = ""
            if (exists < 0)
                errorText = kiko.lastError()
        }
    }

    PullDownMenu {
        MenuItem {
            text: "Settings"
            visible: page.exists === 1
            onClicked: pageStack.push(Qt.resolvedUrl("SettingsPage.qml"), {
                wallet: page.wallet,
                coinPage: page
            })
        }
    }

    SilicaFlickable {
        anchors.fill: parent
        contentHeight: column.height

        Column {
            id: column
            width: parent.width
            spacing: Theme.paddingLarge

            PageHeader { title: kiko.walletName(page.wallet) }

            Label {
                visible: page.errorText !== ""
                width: parent.width - 2 * Theme.horizontalPageMargin
                x: Theme.horizontalPageMargin
                wrapMode: Text.Wrap
                color: Theme.highlightColor
                text: page.errorText
            }

            Column {
                width: parent.width
                spacing: Theme.paddingLarge
                visible: page.exists === 0

                Button {
                    anchors.horizontalCenter: parent.horizontalCenter
                    text: "Create"
                    onClicked: {
                        if (kiko.walletCreate(page.wallet) !== 0)
                            page.errorText = kiko.lastError()
                        else
                            page.refresh()
                    }
                }
                Button {
                    anchors.horizontalCenter: parent.horizontalCenter
                    text: "Restore"
                    onClicked: pageStack.push(Qt.resolvedUrl("RestorePage.qml"), { wallet: page.wallet })
                }
            }

            Column {
                width: parent.width
                spacing: Theme.paddingLarge
                visible: page.exists === 1

                Image {
                    anchors.horizontalCenter: parent.horizontalCenter
                    width: parent.width * 0.62
                    height: width
                    fillMode: Image.PreserveAspectFit
                    source: page.exists === 1 ? "image://kiko/qr/" + page.wallet : ""
                }

                TextArea {
                    width: parent.width
                    readOnly: true
                    text: page.address
                    wrapMode: TextEdit.Wrap
                    label: "Receive"
                }

                Repeater {
                    model: {
                        if (page.exists !== 1)
                            return 0
                        var n = kiko.walletDetailCount(page.wallet)
                        return n < 0 ? 0 : n
                    }
                    delegate: TextArea {
                        width: column.width
                        readOnly: true
                        wrapMode: TextEdit.Wrap
                        label: kiko.walletDetailLabel(page.wallet, index)
                        text: kiko.walletDetailValue(page.wallet, index)
                    }
                }

                Button {
                    anchors.horizontalCenter: parent.horizontalCenter
                    text: "Send"
                    onClicked: pageStack.push(Qt.resolvedUrl("SendPage.qml"), { wallet: page.wallet })
                }
            }
        }
    }
}
