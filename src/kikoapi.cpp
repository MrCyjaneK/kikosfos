#include "kikoapi.h"

#include "../go/libkiko.h"

#include <QByteArray>
#include <QMutex>
#include <QPointer>
#include <QtConcurrent>

namespace {

QByteArray utf8(const QString &text)
{
    return text.toUtf8();
}

QMutex requestGate;

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
    QByteArray pinCopy = pin;
    QByteArray againCopy = again;
    QtConcurrent::run([self, pinCopy, againCopy, setup]() mutable {
        const int rc = setup
            ? KIKO_setup_pin(pinCopy.data(), againCopy.data())
            : KIKO_unlock(pinCopy.data());
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

QString KikoApi::walletTicker(int wallet)
{
    return takeKikoString(KIKO_wallet_ticker(wallet));
}

QString KikoApi::walletWhole(int wallet)
{
    return takeKikoString(KIKO_wallet_whole(wallet));
}

QString KikoApi::walletAtom(int wallet)
{
    return takeKikoString(KIKO_wallet_atom(wallet));
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
    QByteArray text = utf8(mnemonic);
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

QString KikoApi::walletMultisigMaterial(int wallet)
{
    return takeKikoString(KIKO_wallet_multisig_material(wallet));
}

void KikoApi::walletBalance(int wallet)
{
    QPointer<KikoApi> self(this);
    QtConcurrent::run([self, wallet]() {
        const QMutexLocker lock(&requestGate);
        char *text = KIKO_wallet_balance(wallet);
        QString amount;
        QString fiat;
        QString message;
        if (!text) {
            message = takeKikoString(KIKO_last_error());
        } else {
            amount = takeKikoString(text);
            char *fiatText = KIKO_wallet_fiat(wallet);
            if (fiatText)
                fiat = takeKikoString(fiatText);
        }
        if (!self)
            return;
        QMetaObject::invokeMethod(self.data(), "deliverBalance", Qt::QueuedConnection,
                                  Q_ARG(int, wallet), Q_ARG(QString, amount),
                                  Q_ARG(QString, fiat), Q_ARG(QString, message));
    });
}

void KikoApi::deliverBalance(int wallet, const QString &amount, const QString &fiat, const QString &message)
{
    emit balanceReady(wallet, amount, fiat, message);
}

void KikoApi::walletActivity(int wallet)
{
    QPointer<KikoApi> self(this);
    QtConcurrent::run([self, wallet]() {
        const QMutexLocker lock(&requestGate);
        const int rc = KIKO_wallet_activity(wallet);
        QString message;
        if (rc != 0)
            message = takeKikoString(KIKO_last_error());
        if (!self)
            return;
        QMetaObject::invokeMethod(self.data(), "deliverActivity", Qt::QueuedConnection,
                                  Q_ARG(int, wallet), Q_ARG(QString, message));
    });
}

void KikoApi::deliverActivity(int wallet, const QString &message)
{
    emit activityReady(wallet, message);
}

int KikoApi::walletActivityCount(int wallet)
{
    return KIKO_wallet_activity_count(wallet);
}

QString KikoApi::walletActivityDirection(int wallet, int row)
{
    return takeKikoString(KIKO_wallet_activity_direction(wallet, row));
}

QString KikoApi::walletActivityAmount(int wallet, int row)
{
    return takeKikoString(KIKO_wallet_activity_amount(wallet, row));
}

QString KikoApi::walletActivityWhen(int wallet, int row)
{
    return takeKikoString(KIKO_wallet_activity_when(wallet, row));
}

int KikoApi::walletDelete(int wallet, const QString &pin)
{
    QByteArray text = utf8(pin);
    return KIKO_wallet_delete(wallet, text.data());
}

QString KikoApi::walletQuote(int wallet, const QString &amount, const QString &unit)
{
    QByteArray amountUtf8 = utf8(amount);
    QByteArray unitUtf8 = utf8(unit);
    return takeKikoString(KIKO_wallet_quote(wallet, amountUtf8.data(), unitUtf8.data()));
}

void KikoApi::walletWarmEuro(int wallet)
{
    QPointer<KikoApi> self(this);
    QtConcurrent::run([self, wallet]() {
        const QMutexLocker lock(&requestGate);
        KIKO_wallet_warm_euro(wallet);
        if (!self)
            return;
        QMetaObject::invokeMethod(self.data(), "deliverEuro", Qt::QueuedConnection, Q_ARG(int, wallet));
    });
}

void KikoApi::deliverEuro(int wallet)
{
    emit euroReady(wallet);
}

void KikoApi::walletSend(int wallet, const QString &to, const QString &amount)
{
    QPointer<KikoApi> self(this);
    QByteArray toUtf8 = utf8(to);
    QByteArray amountUtf8 = utf8(amount);
    QtConcurrent::run([self, wallet, toUtf8, amountUtf8]() mutable {
        const QMutexLocker lock(&requestGate);
        const int rc = KIKO_wallet_send(wallet, toUtf8.data(), amountUtf8.data());
        QString message;
        if (rc != 0)
            message = takeKikoString(KIKO_last_error());
        if (!self)
            return;
        QMetaObject::invokeMethod(self.data(), "deliverSend", Qt::QueuedConnection,
                                  Q_ARG(int, wallet), Q_ARG(QString, message));
    });
}

void KikoApi::deliverSend(int wallet, const QString &message)
{
    emit sendFinished(wallet, message);
}

QString KikoApi::profileBrand()
{
    return takeKikoString(KIKO_profile_brand());
}

QString KikoApi::profileModel()
{
    return takeKikoString(KIKO_profile_model());
}

QString KikoApi::profileColor()
{
    return takeKikoString(KIKO_profile_color());
}

QString KikoApi::profilePlate()
{
    return takeKikoString(KIKO_profile_plate());
}

QString KikoApi::profileYear()
{
    return takeKikoString(KIKO_profile_year());
}

int KikoApi::profileSaveCar(const QString &brand, const QString &model, const QString &color, const QString &plate, const QString &year)
{
    QByteArray brandUtf8 = utf8(brand);
    QByteArray modelUtf8 = utf8(model);
    QByteArray colorUtf8 = utf8(color);
    QByteArray plateUtf8 = utf8(plate);
    QByteArray yearUtf8 = utf8(year);
    return KIKO_profile_save_car(brandUtf8.data(), modelUtf8.data(), colorUtf8.data(), plateUtf8.data(), yearUtf8.data());
}

int KikoApi::serverCount()
{
    return KIKO_server_count();
}

QString KikoApi::serverHost(int index)
{
    return takeKikoString(KIKO_server_host(index));
}

int KikoApi::serverCompatible(int index)
{
    return KIKO_server_compatible(index);
}

QString KikoApi::serverWalletName(int index)
{
    return takeKikoString(KIKO_server_wallet_name(index));
}

int KikoApi::serverAdd(const QString &host)
{
    QByteArray hostUtf8 = utf8(host);
    return KIKO_server_add(hostUtf8.data());
}

int KikoApi::serverUpdate(int index, const QString &host)
{
    QByteArray hostUtf8 = utf8(host);
    return KIKO_server_update(index, hostUtf8.data());
}

int KikoApi::serverDelete(int index)
{
    return KIKO_server_delete(index);
}

int KikoApi::serverCoin(int index)
{
    return KIKO_server_coin(index);
}

QString KikoApi::serverFeeBase(int index)
{
    return takeKikoString(KIKO_server_fee_base(index));
}

QString KikoApi::serverFeePerKm(int index)
{
    return takeKikoString(KIKO_server_fee_per_km(index));
}

int KikoApi::serverSaveFees(int index, const QString &base, const QString &perKm, const QString &unit)
{
    QByteArray baseUtf8 = utf8(base);
    QByteArray perUtf8 = utf8(perKm);
    QByteArray unitUtf8 = utf8(unit);
    return KIKO_server_save_fees(index, baseUtf8.data(), perUtf8.data(), unitUtf8.data());
}

int KikoApi::serverRefresh()
{
    return KIKO_server_refresh();
}

QString KikoApi::serverChains(int index)
{
    return takeKikoString(KIKO_server_chains(index));
}

int KikoApi::serverSaveChains(int index, const QString &fees)
{
    QByteArray feesUtf8 = utf8(fees);
    return KIKO_server_save_chains(index, feesUtf8.data());
}

void KikoApi::applyFeesEur(const QString &base, const QString &perKm)
{
    QPointer<KikoApi> self(this);
    QByteArray baseUtf8 = utf8(base);
    QByteArray perUtf8 = utf8(perKm);
    QtConcurrent::run([self, baseUtf8, perUtf8]() mutable {
        const QMutexLocker lock(&requestGate);
        const int rc = KIKO_apply_fees_eur(baseUtf8.data(), perUtf8.data());
        QString message;
        if (rc != 0)
            message = takeKikoString(KIKO_last_error());
        if (!self)
            return;
        QMetaObject::invokeMethod(self.data(), "deliverFees", Qt::QueuedConnection,
                                  Q_ARG(QString, message));
    });
}

void KikoApi::deliverFees(const QString &message)
{
    emit feesApplied(message);
}

QString KikoApi::gpsDriver()
{
    return takeKikoString(KIKO_gps_driver());
}

int KikoApi::gpsSetDriver(const QString &kind)
{
    QByteArray kindUtf8 = utf8(kind);
    return KIKO_gps_set_driver(kindUtf8.data());
}

QString KikoApi::gpsPosition()
{
    return takeKikoString(KIKO_gps_position());
}

void KikoApi::gpsRelease()
{
    KIKO_gps_release();
}

void KikoApi::driveRegister()
{
    QPointer<KikoApi> self(this);
    QtConcurrent::run([self]() {
        const QMutexLocker lock(&requestGate);
        const QString message = takeKikoString(KIKO_drive_register());
        if (!self)
            return;
        QMetaObject::invokeMethod(self.data(), "deliverDriveRegistered", Qt::QueuedConnection,
                                  Q_ARG(QString, message));
    });
}

void KikoApi::deliverDriveRegistered(const QString &message)
{
    emit driveRegistered(message);
}

void KikoApi::driveRefresh(double rangeKm)
{
    QPointer<KikoApi> self(this);
    QtConcurrent::run([self, rangeKm]() {
        const QMutexLocker lock(&requestGate);
        const QString message = takeKikoString(KIKO_drive_refresh(rangeKm));
        if (!self)
            return;
        QMetaObject::invokeMethod(self.data(), "deliverDrivesReady", Qt::QueuedConnection,
                                  Q_ARG(QString, message));
    });
}

void KikoApi::deliverDrivesReady(const QString &message)
{
    emit drivesReady(message);
}

int KikoApi::driveCount()
{
    return KIKO_drive_count();
}

QString KikoApi::driveId(int index)
{
    return takeKikoString(KIKO_drive_id(index));
}

QString KikoApi::driveServer(int index)
{
    return takeKikoString(KIKO_drive_server(index));
}

int KikoApi::driveCoin(int index)
{
    return KIKO_drive_coin(index);
}

QString KikoApi::driveStartAddress(int index)
{
    return takeKikoString(KIKO_drive_start_address(index));
}

QString KikoApi::driveEndAddress(int index)
{
    return takeKikoString(KIKO_drive_end_address(index));
}

QString KikoApi::drivePickupKm(int index)
{
    return takeKikoString(KIKO_drive_pickup_km(index));
}

QString KikoApi::driveDistanceKm(int index)
{
    return takeKikoString(KIKO_drive_distance_km(index));
}

QString KikoApi::drivePickupTime(int index)
{
    return takeKikoString(KIKO_drive_pickup_time(index));
}

QString KikoApi::driveFareLovelace(int index)
{
    return takeKikoString(KIKO_drive_fare_lovelace(index));
}

QString KikoApi::driveBaseLovelace(int index)
{
    return takeKikoString(KIKO_drive_base_lovelace(index));
}

QString KikoApi::drivePerKmLovelace(int index)
{
    return takeKikoString(KIKO_drive_per_km_lovelace(index));
}

int KikoApi::driveImmediate(int index)
{
    return KIKO_drive_immediate(index);
}

QString KikoApi::driveWhenDay(int index)
{
    return takeKikoString(KIKO_drive_when_day(index));
}

QString KikoApi::driveWhenClock(int index)
{
    return takeKikoString(KIKO_drive_when_clock(index));
}

QString KikoApi::driveStartLat(int index)
{
    return takeKikoString(KIKO_drive_start_lat(index));
}

QString KikoApi::driveStartLon(int index)
{
    return takeKikoString(KIKO_drive_start_lon(index));
}

QString KikoApi::driveEndLat(int index)
{
    return takeKikoString(KIKO_drive_end_lat(index));
}

QString KikoApi::driveEndLon(int index)
{
    return takeKikoString(KIKO_drive_end_lon(index));
}

void KikoApi::driveNavigate(const QString &lat, const QString &lon, const QString &title)
{
    QPointer<KikoApi> self(this);
    QByteArray latUtf8 = utf8(lat);
    QByteArray lonUtf8 = utf8(lon);
    QByteArray titleUtf8 = utf8(title);
    QtConcurrent::run([self, latUtf8, lonUtf8, titleUtf8]() mutable {
        const QMutexLocker lock(&requestGate);
        const QString message = takeKikoString(KIKO_drive_navigate(latUtf8.data(), lonUtf8.data(), titleUtf8.data()));
        if (!self)
            return;
        QMetaObject::invokeMethod(self.data(), "deliverNavigate", Qt::QueuedConnection,
                                  Q_ARG(QString, message));
    });
}

void KikoApi::deliverNavigate(const QString &message)
{
    emit navigateFinished(message);
}

void KikoApi::driveClaim(const QString &server, const QString &id, const QString &pickupKm, int coin)
{
    QPointer<KikoApi> self(this);
    QByteArray serverUtf8 = utf8(server);
    QByteArray idUtf8 = utf8(id);
    QByteArray kmUtf8 = utf8(pickupKm);
    QtConcurrent::run([self, serverUtf8, idUtf8, kmUtf8, coin]() mutable {
        const QMutexLocker lock(&requestGate);
        const QString message = takeKikoString(KIKO_drive_claim(serverUtf8.data(), idUtf8.data(), kmUtf8.data(), coin));
        if (!self)
            return;
        QMetaObject::invokeMethod(self.data(), "deliverDriveClaimed", Qt::QueuedConnection,
                                  Q_ARG(QString, message));
    });
}

void KikoApi::deliverDriveClaimed(const QString &message)
{
    emit claimFinished(message);
}

void KikoApi::driveVerify(const QString &server, const QString &id)
{
    QPointer<KikoApi> self(this);
    QByteArray serverUtf8 = utf8(server);
    QByteArray idUtf8 = utf8(id);
    QtConcurrent::run([self, serverUtf8, idUtf8]() mutable {
        const QMutexLocker lock(&requestGate);
        const QString text = takeKikoString(KIKO_drive_verify(serverUtf8.data(), idUtf8.data()));
        int seen = 0;
        QString message;
        if (text == QLatin1String("1"))
            seen = 1;
        else if (text != QLatin1String("0"))
            message = text;
        if (!self)
            return;
        QMetaObject::invokeMethod(self.data(), "deliverDriveVerified", Qt::QueuedConnection,
                                  Q_ARG(int, seen), Q_ARG(QString, message));
    });
}

void KikoApi::deliverDriveVerified(int seen, const QString &message)
{
    emit driveVerified(seen, message);
}

void KikoApi::driveCashOut(const QString &server, const QString &id)
{
    QPointer<KikoApi> self(this);
    QByteArray serverUtf8 = utf8(server);
    QByteArray idUtf8 = utf8(id);
    QtConcurrent::run([self, serverUtf8, idUtf8]() mutable {
        const QMutexLocker lock(&requestGate);
        const QString message = takeKikoString(KIKO_drive_cash_out(serverUtf8.data(), idUtf8.data()));
        if (!self)
            return;
        QMetaObject::invokeMethod(self.data(), "deliverCashOut", Qt::QueuedConnection,
                                  Q_ARG(QString, message));
    });
}

void KikoApi::deliverCashOut(const QString &message)
{
    emit cashOutFinished(message);
}

int KikoApi::driveClaimed(const QString &server, const QString &id)
{
    QByteArray serverUtf8 = utf8(server);
    QByteArray idUtf8 = utf8(id);
    return KIKO_drive_claimed(serverUtf8.data(), idUtf8.data());
}

int KikoApi::drivePaid(const QString &server, const QString &id)
{
    QByteArray serverUtf8 = utf8(server);
    QByteArray idUtf8 = utf8(id);
    return KIKO_drive_paid(serverUtf8.data(), idUtf8.data());
}

int KikoApi::driveActiveCount()
{
    return KIKO_drive_active_count();
}

QString KikoApi::driveActive(int index)
{
    return takeKikoString(KIKO_drive_active(index));
}

QString KikoApi::lastError()
{
    return takeKikoString(KIKO_last_error());
}
