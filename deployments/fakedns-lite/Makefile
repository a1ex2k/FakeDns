include $(TOPDIR)/rules.mk

PKG_NAME:=fakedns-lite
PKG_VERSION:=$(if $(VERSION),$(VERSION),0.0.0)
PKG_RELEASE:=1

include $(INCLUDE_DIR)/package.mk

define Package/fakedns-lite
  SECTION:=net
  CATEGORY:=Network
  TITLE:=FakeDNS Core Server
  DEPENDS:=+libstdcpp +nftables-json +kmod-nft-nat
endef

define Package/fakedns-lite/description
 FakeDNS Core Server using nftables DNAT.
endef

define Package/fakedns-lite/conffiles
/etc/config/fakedns-lite
endef

define Build/Prepare
	mkdir -p $(PKG_BUILD_DIR)
endef

define Build/Compile
	true
endef

define Package/fakedns-lite/install
	$(INSTALL_DIR) $(1)/usr/bin
	$(INSTALL_BIN) $(CURDIR)/package/fakedns-lite/files/fakedns-lite $(1)/usr/bin/fakedns-lite

	$(INSTALL_DIR) $(1)/etc/init.d
	$(INSTALL_BIN) $(CURDIR)/package/fakedns-lite/files/fakedns-lite.init $(1)/etc/init.d/fakedns-lite

	$(INSTALL_DIR) $(1)/etc/config
	$(INSTALL_CONF) $(CURDIR)/package/fakedns-lite/files/fakedns-lite.config $(1)/etc/config/fakedns-lite
endef

define Package/fakedns-lite/postinst
#!/bin/sh
[ -n "$$IPKG_INSTROOT" ] && exit 0
/etc/init.d/fakedns-lite enable
/etc/init.d/fakedns-lite restart
exit 0
endef

define Package/fakedns-lite/prerm
#!/bin/sh
[ -n "$$IPKG_INSTROOT" ] && exit 0
/etc/init.d/fakedns-lite stop
/etc/init.d/fakedns-lite disable
exit 0
endef

$(eval $(call BuildPackage,fakedns-lite))
