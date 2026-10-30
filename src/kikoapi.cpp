#include "kikoapi.h"

#include "../go/libkiko.h"

#include <QByteArray>
#include <QPointer>
#include <QtConcurrent>

namespace {

QByteArray utf8(const QString &text)
{
    return text.toUtf8();
}

}

QString takeKikoString(char *text)
{
    const QString value = QString::fromUtf8(text ? text : "");
    if (text)
        KIKO_free(text);
    return value;
}

KikoApi::KikoApi(QObject *parent)
    : QObject(parent)
{
}

int KikoApi::pinIsSet()
{
    return KIKO_pin_is_set();
}

void KikoApi::setupPin(const QString &pin, const QString &again)
{
    runPin(utf8(pin), utf8(again), true);
}

void KikoApi::unlockPin(const QString &pin)
{
    runPin(utf8(pin), QByteArray(), false);
}

void KikoApi::runPin(const QByteArray &pin, const QByteArray &again, bool setup)
{
    QPointer<KikoApi> self(this);
    QtConcurrent::run([self, pin, again, setup]() mutable {
        const int rc = setup
            ? KIKO_setup_pin(pin.data(), again.data())
            : KIKO_unlock(pin.data());
        QString message;
        if (rc != 0)
            message = takeKikoString(KIKO_last_error());
        if (!self)
            return;
        QMetaObject::invokeMethod(self.data(), "deliverPin", Qt::QueuedConnection,
                                  Q_ARG(int, rc), Q_ARG(QString, message));
    });
}

void KikoApi::deliverPin(int rc, const QString &message)
{
    if (rc == 0)
        emit pinAccepted();
    else
        emit pinRejected(message);
}

int KikoApi::walletCount()
{
    return KIKO_wallet_count();
}

QString KikoApi::walletName(int wallet)
{
    return takeKikoString(KIKO_wallet_name(wallet));
}

int KikoApi::walletExists(int wallet)
{
    return KIKO_wallet_exists(wallet);
}

QString KikoApi::walletAddress(int wallet)
{
    return takeKikoString(KIKO_wallet_address(wallet));
}

int KikoApi::walletDetailCount(int wallet)
{
    return KIKO_wallet_detail_count(wallet);
}

QString KikoApi::walletDetailLabel(int wallet, int detail)
{
    return takeKikoString(KIKO_wallet_detail_label(wallet, detail));
}

QString KikoApi::walletDetailValue(int wallet, int detail)
{
    return takeKikoString(KIKO_wallet_detail_value(wallet, detail));
}

int KikoApi::walletCreate(int wallet)
{
    return KIKO_wallet_create(wallet);
}

int KikoApi::walletRestore(int wallet, const QString &mnemonic)
{
    const QByteArray text = utf8(mnemonic);
    return KIKO_wallet_restore(wallet, text.data());
}

QString KikoApi::walletMnemonic(int wallet)
{
    return takeKikoString(KIKO_wallet_mnemonic(wallet));
}

QString KikoApi::walletPrivateKey(int wallet)
{
    return takeKikoString(KIKO_wallet_private_key(wallet));
}

int KikoApi::walletDelete(int wallet, const QString &pin)
{
    const QByteArray text = utf8(pin);
    return KIKO_wallet_delete(wallet, text.data());
}

int KikoApi::walletSend(int wallet, const QString &to, const QString &amount)
{
    const QByteArray toUtf8 = utf8(to);
    const QByteArray amountUtf8 = utf8(amount);
    return KIKO_wallet_send(wallet, toUtf8.data(), amountUtf8.data());
}

QString KikoApi::lastError()
{
    return takeKikoString(KIKO_last_error());
}
