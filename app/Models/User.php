<?php

namespace App\Models;

// use Illuminate\Contracts\Auth\MustVerifyEmail;
use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Foundation\Auth\User as Authenticatable;
use Illuminate\Notifications\Notifiable;
use Laravel\Passport\HasApiTokens;

class User extends Authenticatable
{
    /** @use HasFactory<\Database\Factories\UserFactory> */
    use HasApiTokens, HasFactory, Notifiable;

    /**
     * The attributes that are mass assignable.
     *
     * @var list<string>
     */
    protected $fillable = [
        'name',
        'email',
        'mobile',
        'password',
    ];

    /**
     * The attributes that should be hidden for serialization.
     *
     * @var list<string>
     */
    protected $hidden = [
        'password',
        'remember_token',
    ];

    /**
     * Get the attributes that should be cast.
     *
     * @return array<string, string>
     */
    protected function casts(): array
    {
        return [
            'email_verified_at' => 'datetime',
            'mobile_verified_at' => 'datetime',
            'password' => 'hashed',
        ];
    }

    /**
     * Get the user's profile
     */
    public function profile(): \Illuminate\Database\Eloquent\Relations\HasOne
    {
        return $this->hasOne(UserProfile::class);
    }

    /**
     * Get the user's daily health logs
     */
    public function dailyHealthLogs(): \Illuminate\Database\Eloquent\Relations\HasMany
    {
        return $this->hasMany(DailyHealthLog::class);
    }

    /**
     * Get the user's cycle calculations
     */
    public function cycleCalculations(): \Illuminate\Database\Eloquent\Relations\HasMany
    {
        return $this->hasMany(CycleCalculation::class);
    }

    /**
     * Get the user's cycle history
     */
    public function cycleHistories(): \Illuminate\Database\Eloquent\Relations\HasMany
    {
        return $this->hasMany(CycleHistory::class);
    }

    /**
     * Get the user's pregnancy profile
     */
    public function pregnancyProfile(): \Illuminate\Database\Eloquent\Relations\HasOne
    {
        return $this->hasOne(PregnancyProfile::class);
    }

    /**
     * Get the user's pregnancy symptom logs
     */
    public function pregnancySymptomLogs(): \Illuminate\Database\Eloquent\Relations\HasMany
    {
        return $this->hasMany(PregnancySymptomLog::class);
    }

    /**
     * Get the user's pregnancy weekly logs
     */
    public function pregnancyWeeklyLogs(): \Illuminate\Database\Eloquent\Relations\HasMany
    {
        return $this->hasMany(PregnancyWeeklyLog::class);
    }

    /**
     * Get the user's pregnancy fetal movements
     */
    public function pregnancyFetalMovements(): \Illuminate\Database\Eloquent\Relations\HasMany
    {
        return $this->hasMany(PregnancyFetalMovement::class);
    }

    /**
     * Get the user's pregnancy alerts
     */
    public function pregnancyAlerts(): \Illuminate\Database\Eloquent\Relations\HasMany
    {
        return $this->hasMany(PregnancyAlert::class);
    }
}
