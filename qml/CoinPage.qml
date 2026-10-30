import QtQuick 2.6
import Sailfish.Silica 1.0
import "components"

Page {
    id: page
    objectName: "coinPage"
    property int wallet
    property int exists: -1
    property string address
    property string errorText
    property string balance
    property string fiat
    property string balanceMessage
    property int balanceState: 0
    property int activityCount: 0
    property int activityState: 0
    property string activityMessage
    property string shownAddress

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
                balance = ""
                fiat = ""
                balanceMessage = ""
                balanceState = 0
                activityCount = 0
                activityMessage = ""
                activityState = 0
                shownAddress = ""
            } else {
                if (address !== shownAddress) {
                    shownAddress = address
                    balance = ""
                    fiat = ""
                    activityCount = 0
                    activityMessage = ""
                    activityState = 0
                }
                if (balance === "")
                    balanceState = 0
                if (activityCount === 0)
                    activityState = 0
                kiko.walletBalance(wallet)
                kiko.walletActivity(wallet)
            }
        } else {
            address = ""
            balance = ""
            fiat = ""
            balanceMessage = ""
            balanceState = 0
            activityCount = 0
            activityMessage = ""
            activityState = 0
            shownAddress = ""
            if (exists < 0)
                errorText = kiko.lastError()
        }
    }

    Connections {
        target: kiko
        onBalanceReady: {
            if (wallet !== page.wallet || page.exists !== 1)
                return
            page.balanceState = message === "" ? 1 : -1
            page.balance = amount
            if (fiat !== "")
                page.fiat = fiat
            page.balanceMessage = message
        }
        onActivityReady: {
            if (wallet !== page.wallet || page.exists !== 1)
                return
            page.activityMessage = message
            page.activityState = 1
            if (message === "")
                page.activityCount = kiko.walletActivityCount(page.wallet)
        }
    }

    SilicaFlickable {
        anchors.fill: parent
        contentHeight: Math.max(column.height, height)

        PullDownMenu {
            MenuItem {
                text: "Send"
                visible: page.exists === 1
                onClicked: pageStack.push(Qt.resolvedUrl("SendPage.qml"), { wallet: page.wallet })
            }
            MenuItem {
                text: "Settings"
                visible: page.exists === 1
                onClicked: pageStack.push(Qt.resolvedUrl("SettingsPage.qml"), {
                    wallet: page.wallet,
                    coinPage: page
                })
            }
        }

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

                Column {
                    width: parent.width
                    spacing: Theme.paddingSmall

                    Label {
                        anchors.horizontalCenter: parent.horizontalCenter
                        text: "Balance"
                        color: Theme.secondaryColor
                        font.pixelSize: Theme.fontSizeSmall
                    }

                    BusyIndicator {
                        anchors.horizontalCenter: parent.horizontalCenter
                        running: page.balanceState === 0
                        visible: running
                        width: Theme.itemSizeSmall
                        height: width
                    }

                    Label {
                        anchors.horizontalCenter: parent.horizontalCenter
                        visible: page.balance !== ""
                        text: page.balance
                        color: Theme.primaryColor
                        font.pixelSize: Theme.fontSizeExtraLarge
                    }

                    Label {
                        anchors.horizontalCenter: parent.horizontalCenter
                        visible: page.fiat !== ""
                        text: page.fiat
                        color: Theme.secondaryColor
                        font.pixelSize: Theme.fontSizeLarge
                    }

                    Label {
                        width: parent.width - 2 * Theme.horizontalPageMargin
                        anchors.horizontalCenter: parent.horizontalCenter
                        horizontalAlignment: Text.AlignHCenter
                        wrapMode: Text.Wrap
                        visible: page.balanceMessage !== ""
                        text: page.balanceMessage
                        color: Theme.highlightColor
                        font.pixelSize: Theme.fontSizeSmall
                    }
                }

                Image {
                    anchors.horizontalCenter: parent.horizontalCenter
                    width: parent.width * 0.62
                    height: width
                    fillMode: Image.PreserveAspectFit
                    source: page.exists === 1 ? "image://kiko/qr/" + page.wallet : ""
                }

                SecretField {
                    label: "Receive"
                    value: page.address
                }

                Repeater {
                    model: {
                        if (page.exists !== 1)
                            return 0
                        var n = kiko.walletDetailCount(page.wallet)
                        return n < 0 ? 0 : n
                    }
                    delegate: SecretField {
                        label: kiko.walletDetailLabel(page.wallet, index)
                        value: kiko.walletDetailValue(page.wallet, index)
                    }
                }

                Column {
                    width: parent.width
                    spacing: Theme.paddingSmall

                    Label {
                        x: Theme.horizontalPageMargin
                        text: "Activity"
                        color: Theme.highlightColor
                        font.pixelSize: Theme.fontSizeLarge
                    }

                    BusyIndicator {
                        anchors.horizontalCenter: parent.horizontalCenter
                        running: page.activityState === 0
                        visible: running
                        width: Theme.itemSizeSmall
                        height: width
                    }

                    Label {
                        x: Theme.horizontalPageMargin
                        visible: page.activityMessage !== ""
                        width: parent.width - 2 * Theme.horizontalPageMargin
                        wrapMode: Text.Wrap
                        text: page.activityMessage
                        color: Theme.highlightColor
                        font.pixelSize: Theme.fontSizeSmall
                    }

                    Label {
                        x: Theme.horizontalPageMargin
                        visible: page.activityState === 1 && page.activityCount === 0 && page.activityMessage === ""
                        text: "No transactions"
                        color: Theme.secondaryColor
                    }
                }

                Repeater {
                    model: page.activityCount
                    delegate: Column {
                        width: parent.width
                        spacing: Theme.paddingSmall

                        Item {
                            width: parent.width
                            height: directionLabel.height

                            Label {
                                id: directionLabel
                                x: Theme.horizontalPageMargin
                                text: kiko.walletActivityDirection(page.wallet, index)
                                color: Theme.primaryColor
                            }

                            Label {
                                anchors.right: parent.right
                                anchors.rightMargin: Theme.horizontalPageMargin
                                anchors.verticalCenter: directionLabel.verticalCenter
                                text: kiko.walletActivityAmount(page.wallet, index)
                                color: Theme.highlightColor
                            }
                        }

                        Label {
                            x: Theme.horizontalPageMargin
                            text: kiko.walletActivityWhen(page.wallet, index)
                            color: Theme.secondaryColor
                            font.pixelSize: Theme.fontSizeSmall
                        }
                    }
                }

            }
        }
    }
}
