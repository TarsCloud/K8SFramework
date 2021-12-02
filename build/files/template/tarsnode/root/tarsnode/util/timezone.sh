#!/usr/bin/env bash

if cd /usr/share/zoneinfo; then

  _LOCALTIME_FILE_="/etc/localtime"

  if [ -f "${_LOCALTIME_FILE_}" ]; then

    _ZONE_FILES_=$(tail -c 6 "${_LOCALTIME_FILE_}" | xargs -I % grep -arl '%$' --exclude-dir right --exclude-dir posix)

    _ZONE_FILES_ARRAY_=(${_ZONE_FILES_})

    _LOCALTIME_ZONE_MD5_=$(zdump -v "${_LOCALTIME_FILE_}" | awk '{print $2$3$4$6$7$8$9$10$11$12$13$14$15$16}' | md5sum)
    for ZONEFILE in "${_ZONE_FILES_ARRAY_[@]}"; do
      _MD5_=$(zdump -v "${ZONEFILE}" | awk '{print $2$3$4$6$7$8$9$10$11$12$13$14$15$16}' | md5sum)
      if [ "$_MD5_" = "$_LOCALTIME_ZONE_MD5_" ]; then
        echo "Will Update TimeZone To" "${ZONEFILE}"
        rm -rf "${_LOCALTIME_FILE_}"
        ln -sf /usr/share/zoneinfo/"${ZONEFILE}" /etc/localtime
        echo "${ZONEFILE}" >"${_LOCALTIME_FILE_}"
        export TZ=${ZONEFILE}
        break
      fi
    done
  fi
fi
