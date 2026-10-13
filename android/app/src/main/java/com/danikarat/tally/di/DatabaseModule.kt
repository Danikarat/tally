package com.danikarat.tally.di

import android.content.Context
import androidx.room.Room
import com.danikarat.tally.data.local.TallyDatabase
import com.danikarat.tally.data.local.TransactionDao
import dagger.Module
import dagger.Provides
import dagger.hilt.InstallIn
import dagger.hilt.android.qualifiers.ApplicationContext
import dagger.hilt.components.SingletonComponent
import javax.inject.Singleton

/**
 * DatabaseModule provides Room database and DAO singletons for Hilt injection.
 */
@Module
@InstallIn(SingletonComponent::class)
object DatabaseModule {

    private const val DATABASE_NAME = "tally.db"

    @Provides
    @Singleton
    fun provideDatabase(
        @ApplicationContext context: Context
    ): TallyDatabase {
        return Room.databaseBuilder(
            context,
            TallyDatabase::class.java,
            DATABASE_NAME
        ).fallbackToDestructiveMigration().build()
    }

    @Provides
    @Singleton
    fun provideTransactionDao(
        database: TallyDatabase
    ): TransactionDao {
        return database.transactionDao()
    }
}
