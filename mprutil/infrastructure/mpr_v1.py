from __future__ import annotations
import sqlite3
from typing import Iterator

from mprutil.domain.unit import Unit, UnitId


class MprV1Reader:
    """SQLite-backed MPR v1 reader"""

    def __init__(self, path: str) -> None:
        self._path = path
        self._conn: sqlite3.Connection | None = None

    def _db(self) -> sqlite3.Connection:
        if self._conn is None:
            self._conn = sqlite3.connect(self._path)
            self._conn.row_factory = sqlite3.Row
        return self._conn

    def iter_units(self, type_filter: str | None = None) -> Iterator[Unit]:
        cur = self._db().execute(
            "SELECT UnitID, ContainerID, ContainmentName, Contents FROM Unit"
        )
        for row in cur:
            uid_raw = row["UnitID"]
            contents = row["Contents"]
            if not contents:
                continue
            uid = UnitId.guid_le_to_uuid(uid_raw) if len(uid_raw) == 16 else uid_raw.hex()
            if type_filter and type_filter not in contents:
                continue
            yield Unit(
                id=UnitId(uid),
                container_id=row["ContainerID"],
                containment_name=row["ContainmentName"] or "",
                raw_bytes=contents,
            )

    def get_unit(self, uid: str | UnitId) -> Unit | None:
        if isinstance(uid, UnitId):
            uid_str = uid.uuid
        else:
            uid_str = uid
        guid_le = UnitId.uuid_to_guid_le(uid_str)
        cur = self._db().execute(
            "SELECT UnitID, ContainerID, ContainmentName, Contents FROM Unit WHERE UnitID = ?",
            (guid_le,),
        )
        row = cur.fetchone()
        if row is None or not row["Contents"]:
            return None
        return Unit(
            id=UnitId(uid_str),
            container_id=row["ContainerID"],
            containment_name=row["ContainmentName"] or "",
            raw_bytes=row["Contents"],
        )

    @property
    def project_version(self) -> str:
        cur = self._db().execute("SELECT _ProductVersion FROM _MetaData")
        row = cur.fetchone()
        return row[0] if row else "?"

    def close(self) -> None:
        if self._conn:
            self._conn.close()
            self._conn = None

    def __enter__(self) -> MprV1Reader:
        return self

    def __exit__(self, *args: object) -> None:
        self.close()
