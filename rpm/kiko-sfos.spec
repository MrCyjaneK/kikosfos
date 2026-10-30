Name:       kiko-sfos
Summary:    Kiko for Sailfish
Version:    0.1.0
Release:    1
License:    BSD-3-Clause
Source0:    %{name}-%{version}.tar.bz2
BuildRequires:  pkgconfig(Qt5Core)
BuildRequires:  pkgconfig(Qt5Gui)
BuildRequires:  pkgconfig(Qt5Qml)
BuildRequires:  pkgconfig(Qt5Quick)
BuildRequires:  pkgconfig(Qt5Concurrent)
BuildRequires:  pkgconfig(sailfishapp)
Requires:       sailfishsilica-qt5

%description
Kiko for Sailfish.

%prep
%setup -q -n %{name}-%{version}

%build
CGO_ENABLED=1 go build -C go -buildmode=c-shared -ldflags '-s -w' -o libkiko.so .
CGO_ENABLED=1 go build -C gocar -mod=vendor -buildmode=c-shared -ldflags '-s -w' -o libcargadgetd_app_gocar.so .
%qmake5
%make_build

%install
%qmake5_install
install -Dm 644 sailjail/PureMaps.permission %{buildroot}%{_sysconfdir}/sailjail/permissions/PureMaps.permission
install -Dm 755 gocar/libcargadgetd_app_gocar.so %{buildroot}/usr/lib/cargadgetd/plugins/libcargadgetd_app_gocar.so

%files
%license LICENSES/BSD-3-Clause.txt
%{_bindir}/kiko-sfos
%{_datadir}/kiko-sfos/qml
%{_datadir}/applications/kiko-sfos.desktop
%{_datadir}/icons/hicolor/*/apps/kiko-sfos.png
%{_libdir}/libkiko.so
/usr/lib/cargadgetd/plugins/libcargadgetd_app_gocar.so
%{_sysconfdir}/sailjail/permissions/PureMaps.permission
