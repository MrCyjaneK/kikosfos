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
            property string amountText
            property string fiatText

            function reload() {
                exists = kiko.walletExists(index)
                if (exists === 1)
                    kiko.walletBalance(index)
                else {
                    amountText = ""
                    fiatText = ""
                }
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
                    if (message !== "") {
                        row.amountText = message
                        row.fiatText = ""
                    } else {
                        row.amountText = amount
                        row.fiatText = fiat
                    }
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

                Item {
                    width: parent.width
                    height: amountLabel.implicitHeight
                    visible: row.amountText !== "" || row.fiatText !== ""

                    Label {
                        id: amountLabel
                        anchors.left: parent.left
                        anchors.right: fiatLabel.visible ? fiatLabel.left : parent.right
                        anchors.rightMargin: fiatLabel.visible ? Theme.paddingMedium : 0
                        text: row.amountText
                        font.pixelSize: Theme.fontSizeSmall
                        color: Theme.secondaryColor
                        truncationMode: TruncationMode.Fade
                    }

                    Label {
                        id: fiatLabel
                        anchors.right: parent.right
                        anchors.baseline: amountLabel.baseline
                        visible: row.fiatText !== ""
                        text: row.fiatText
                        font.pixelSize: Theme.fontSizeExtraSmall
                        color: Theme.secondaryColor
                    }
                }
            }

            Image {
                id: statusMark
                anchors.right: parent.right
                anchors.rightMargin: Theme.horizontalPageMargin
                anchors.verticalCenter: parent.verticalCenter
                width: Theme.iconSizeSmall
                height: width
                visible: row.exists >= 0
                source: row.exists === 1 ? "image://theme/icon-m-accept" : "image://theme/icon-m-cancel"
            }

            onClicked: pageStack.push(Qt.resolvedUrl("CoinPage.qml"), { wallet: index })
        }
    }
}
