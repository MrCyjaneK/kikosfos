import QtQuick 2.6
import Sailfish.Silica 1.0

Page {
    id: walletsPage

    property int refreshStamp: 0

    onStatusChanged: {
        if (status === PageStatus.Active)
            refreshStamp++
    }

    SilicaListView {
        anchors.fill: parent
        header: PageHeader { title: "Wallets" }
        model: kiko.walletCount()

        delegate: ListItem {
            id: row
            contentHeight: Theme.itemSizeLarge
            property int exists: -1
            property string balance

            function reload() {
                exists = kiko.walletExists(index)
                if (exists === 1)
                    kiko.walletBalance(index)
                else
                    balance = ""
            }

            Component.onCompleted: reload()

            Connections {
                target: walletsPage
                onRefreshStampChanged: row.reload()
            }

            Connections {
                target: kiko
                onBalanceReady: {
                    if (wallet !== index || row.exists !== 1)
                        return
                    row.balance = message !== "" ? message : amount
                }
            }

            Image {
                id: icon
                anchors.left: parent.left
                anchors.leftMargin: Theme.horizontalPageMargin
                anchors.verticalCenter: parent.verticalCenter
                width: Theme.iconSizeMedium
                height: Theme.iconSizeMedium
                source: "image://kiko/icon/" + index
                fillMode: Image.PreserveAspectFit
            }

            Column {
                anchors.left: icon.right
                anchors.leftMargin: Theme.paddingMedium
                anchors.right: statusMark.left
                anchors.rightMargin: Theme.paddingMedium
                anchors.verticalCenter: parent.verticalCenter

                Label {
                    width: parent.width
                    truncationMode: TruncationMode.Fade
                    text: kiko.walletName(index)
                }

                Label {
                    width: parent.width
                    visible: row.balance !== ""
                    truncationMode: TruncationMode.Fade
                    text: row.balance
                    color: Theme.secondaryColor
                    font.pixelSize: Theme.fontSizeSmall
                }
            }

            Label {
                id: statusMark
                anchors.right: parent.right
                anchors.rightMargin: Theme.horizontalPageMargin
                anchors.verticalCenter: parent.verticalCenter
                visible: row.exists >= 0
                text: row.exists === 1 ? "✓" : "✗"
                color: row.exists === 1 ? "#3DDC84" : "#FF5252"
                font.pixelSize: Theme.fontSizeLarge
            }

            onClicked: pageStack.push(Qt.resolvedUrl("CoinPage.qml"), { wallet: index })
        }
    }
}
