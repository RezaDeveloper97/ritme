package ir.ritmeapp.ritme.adapter.outbound.persistence

import android.content.Context
import android.database.sqlite.SQLiteDatabase
import android.database.sqlite.SQLiteOpenHelper

/**
 * The app's single SQLite database, opened directly via [SQLiteOpenHelper] (CLAUDE.md §3 —
 * no Room). Tables are added here as features need local cache/offline storage; today it
 * holds only the crash-recovery "last safe screen" row.
 */
class RitmeSqliteHelper(
    context: Context,
) : SQLiteOpenHelper(context.applicationContext, DB_NAME, null, DB_VERSION) {

    override fun onCreate(db: SQLiteDatabase) {
        db.execSQL(
            "CREATE TABLE $TABLE_SAFE_SCREEN (" +
                "$COL_ID INTEGER PRIMARY KEY, " +
                "$COL_ROUTE TEXT NOT NULL, " +
                "$COL_ARGS TEXT, " +
                "$COL_TIMESTAMP INTEGER NOT NULL)",
        )
    }

    override fun onUpgrade(db: SQLiteDatabase, oldVersion: Int, newVersion: Int) {
        // Safe-screen data is a disposable cache; rebuilding it on upgrade is acceptable.
        db.execSQL("DROP TABLE IF EXISTS $TABLE_SAFE_SCREEN")
        onCreate(db)
    }

    companion object {
        const val DB_NAME = "ritme.db"
        const val DB_VERSION = 1

        const val TABLE_SAFE_SCREEN = "safe_screen"
        const val COL_ID = "id"
        const val COL_ROUTE = "route"
        const val COL_ARGS = "args"
        const val COL_TIMESTAMP = "ts"

        /** The single-row id; the safe screen is a singleton record we upsert in place. */
        const val SINGLE_ROW_ID = 1
    }
}
