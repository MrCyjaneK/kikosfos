#include "kikoapi.h"
#include "kiko_abi.h"

#include <QString>

#include <cstdlib>

KikoApi::KikoApi(QObject *parent)
    : QObject(parent)
{
}

QString KikoApi::hello() const
{
    char *text = KIKOAPI_hello();
    const QString value = QString::fromUtf8(text ? text : "");
    std::free(text);
    return value;
}
