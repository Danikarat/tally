package com.danikarat.tally.data.local

import androidx.room.Database
import androidx.room.RoomDatabase
import com.danikarat.tally.data.local.entity.TransactionEntity

/**
 * TallyDatabase is the Room SQLite database for Tally offline storage.
 */
@Database(
    entities = [TransactionEntity::class],
    version = 1,
    exportSchema = false
)
abstract class TallyDatabase : RoomDatabase() {
    abstract fun transactionDao(): TransactionDao
}
