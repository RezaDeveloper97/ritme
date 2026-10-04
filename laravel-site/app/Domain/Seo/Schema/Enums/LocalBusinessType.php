<?php

declare(strict_types=1);

namespace App\Domain\Seo\Schema\Enums;

/**
 * LocalBusiness subtypes the mother & child directory uses (pick the most specific one).
 */
enum LocalBusinessType: string
{
    case LocalBusiness = 'LocalBusiness';
    case ChildCare = 'ChildCare';
    case SportsActivityLocation = 'SportsActivityLocation';
    case ExerciseGym = 'ExerciseGym';
    case MedicalClinic = 'MedicalClinic';
    case Physician = 'Physician';
    case Dentist = 'Dentist';
    case Pharmacy = 'Pharmacy';
    case HealthAndBeautyBusiness = 'HealthAndBeautyBusiness';
    case DaySpa = 'DaySpa';
    case Store = 'Store';
    case ClothingStore = 'ClothingStore';
    case ToyStore = 'ToyStore';
}
