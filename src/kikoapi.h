#pragma once

#include <QByteArray>
#include <QObject>
#include <QString>

class KikoApi : public QObject
{
    Q_OBJECT
public:
    explicit KikoApi(QObject *parent = nullptr);

    Q_INVOKABLE int pinIsSet();
    Q_INVOKABLE void setupPin(const QString &pin, const QString &again);
    Q_INVOKABLE void unlockPin(const QString &pin);

    Q_INVOKABLE int walletCount();
    Q_INVOKABLE QString walletName(int wallet);
    Q_INVOKABLE int walletExists(int wallet);
    Q_INVOKABLE QString walletAddress(int wallet);
    Q_INVOKABLE int walletDetailCount(int wallet);
    Q_INVOKABLE QString walletDetailLabel(int wallet, int detail);
    Q_INVOKABLE QString walletDetailValue(int wallet, int detail);
    Q_INVOKABLE int walletCreate(int wallet);
    Q_INVOKABLE int walletRestore(int wallet, const QString &mnemonic);
    Q_INVOKABLE QString walletMnemonic(int wallet);
    Q_INVOKABLE QString walletPrivateKey(int wallet);
    Q_INVOKABLE int walletDelete(int wallet, const QString &pin);
    Q_INVOKABLE int walletSend(int wallet, const QString &to, const QString &amount);
    Q_INVOKABLE QString lastError();

signals:
    void pinAccepted();
    void pinRejected(const QString &message);

private slots:
    void deliverPin(int rc, const QString &message);

private:
    void runPin(const QByteArray &pin, const QByteArray &again, bool setup);
};

QString takeKikoString(char *text);
