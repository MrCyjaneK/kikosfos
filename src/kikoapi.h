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
    Q_INVOKABLE QString walletTicker(int wallet);
    Q_INVOKABLE QString walletWhole(int wallet);
    Q_INVOKABLE QString walletAtom(int wallet);
    Q_INVOKABLE int walletExists(int wallet);
    Q_INVOKABLE QString walletAddress(int wallet);
    Q_INVOKABLE int walletDetailCount(int wallet);
    Q_INVOKABLE QString walletDetailLabel(int wallet, int detail);
    Q_INVOKABLE QString walletDetailValue(int wallet, int detail);
    Q_INVOKABLE int walletCreate(int wallet);
    Q_INVOKABLE int walletRestore(int wallet, const QString &mnemonic);
    Q_INVOKABLE QString walletMnemonic(int wallet);
    Q_INVOKABLE QString walletPrivateKey(int wallet);
    Q_INVOKABLE QString walletMultisigMaterial(int wallet);
    Q_INVOKABLE void walletBalance(int wallet);
    Q_INVOKABLE void walletActivity(int wallet);
    Q_INVOKABLE int walletActivityCount(int wallet);
    Q_INVOKABLE QString walletActivityDirection(int wallet, int row);
    Q_INVOKABLE QString walletActivityAmount(int wallet, int row);
    Q_INVOKABLE QString walletActivityWhen(int wallet, int row);
    Q_INVOKABLE int walletDelete(int wallet, const QString &pin);
    Q_INVOKABLE QString walletQuote(int wallet, const QString &amount, const QString &unit);
    Q_INVOKABLE void walletWarmEuro(int wallet);
    Q_INVOKABLE void walletSend(int wallet, const QString &to, const QString &amount);
    Q_INVOKABLE QString profileBrand();
    Q_INVOKABLE QString profileModel();
    Q_INVOKABLE QString profileColor();
    Q_INVOKABLE QString profilePlate();
    Q_INVOKABLE QString profileYear();
    Q_INVOKABLE int profileSaveCar(const QString &brand, const QString &model, const QString &color, const QString &plate, const QString &year);
    Q_INVOKABLE int serverCount();
    Q_INVOKABLE QString serverHost(int index);
    Q_INVOKABLE int serverCompatible(int index);
    Q_INVOKABLE QString serverWalletName(int index);
    Q_INVOKABLE int serverAdd(const QString &host);
    Q_INVOKABLE int serverUpdate(int index, const QString &host);
    Q_INVOKABLE int serverDelete(int index);
    Q_INVOKABLE int serverCoin(int index);
    Q_INVOKABLE QString serverFeeBase(int index);
    Q_INVOKABLE QString serverFeePerKm(int index);
    Q_INVOKABLE int serverSaveFees(int index, const QString &base, const QString &perKm, const QString &unit);
    Q_INVOKABLE void applyFeesEur(const QString &base, const QString &perKm);
    Q_INVOKABLE QString gpsDriver();
    Q_INVOKABLE int gpsSetDriver(const QString &kind);
    Q_INVOKABLE QString gpsPosition();
    Q_INVOKABLE void gpsRelease();
    Q_INVOKABLE void driveRegister();
    Q_INVOKABLE void driveRefresh(double rangeKm);
    Q_INVOKABLE int driveCount();
    Q_INVOKABLE QString driveId(int index);
    Q_INVOKABLE QString driveServer(int index);
    Q_INVOKABLE int driveCoin(int index);
    Q_INVOKABLE QString driveStartAddress(int index);
    Q_INVOKABLE QString driveEndAddress(int index);
    Q_INVOKABLE QString drivePickupKm(int index);
    Q_INVOKABLE QString driveDistanceKm(int index);
    Q_INVOKABLE QString drivePickupTime(int index);
    Q_INVOKABLE QString driveFareLovelace(int index);
    Q_INVOKABLE QString driveBaseLovelace(int index);
    Q_INVOKABLE QString drivePerKmLovelace(int index);
    Q_INVOKABLE int driveImmediate(int index);
    Q_INVOKABLE QString driveWhenDay(int index);
    Q_INVOKABLE QString driveWhenClock(int index);
    Q_INVOKABLE QString driveStartLat(int index);
    Q_INVOKABLE QString driveStartLon(int index);
    Q_INVOKABLE QString driveEndLat(int index);
    Q_INVOKABLE QString driveEndLon(int index);
    Q_INVOKABLE void driveNavigate(const QString &lat, const QString &lon, const QString &title);
    Q_INVOKABLE void driveClaim(const QString &server, const QString &id, const QString &pickupKm);
    Q_INVOKABLE void driveVerify(const QString &server, const QString &id);
    Q_INVOKABLE int driveClaimed(const QString &server, const QString &id);
    Q_INVOKABLE int drivePaid(const QString &server, const QString &id);
    Q_INVOKABLE QString lastError();

signals:
    void pinAccepted();
    void pinRejected(const QString &message);
    void balanceReady(int wallet, const QString &amount, const QString &fiat, const QString &message);
    void activityReady(int wallet, const QString &message);
    void euroReady(int wallet);
    void sendFinished(int wallet, const QString &message);
    void feesApplied(const QString &message);
    void driveRegistered(const QString &message);
    void drivesReady(const QString &message);
    void navigateFinished(const QString &message);
    void claimFinished(const QString &message);
    void driveVerified(int seen, const QString &message);

private slots:
    void deliverPin(int rc, const QString &message);
    void deliverBalance(int wallet, const QString &amount, const QString &fiat, const QString &message);
    void deliverActivity(int wallet, const QString &message);
    void deliverEuro(int wallet);
    void deliverSend(int wallet, const QString &message);
    void deliverFees(const QString &message);
    void deliverDriveRegistered(const QString &message);
    void deliverDrivesReady(const QString &message);
    void deliverNavigate(const QString &message);
    void deliverDriveClaimed(const QString &message);
    void deliverDriveVerified(int seen, const QString &message);

private:
    void runPin(const QByteArray &pin, const QByteArray &again, bool setup);
};

QString takeKikoString(char *text);
