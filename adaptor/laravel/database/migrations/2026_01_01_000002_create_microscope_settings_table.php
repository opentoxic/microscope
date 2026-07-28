<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        if (Schema::getConnection()->getDriverName() !== 'pgsql') {
            return;
        }

        if (Schema::hasTable('microscope_settings')) {
            return;
        }

        Schema::create('microscope_settings', function (Blueprint $table) {
            $table->string('type')->primary();
            $table->boolean('enabled')->default(true);
            $table->timestampTz('updated_at')->useCurrent();
        });
    }

    public function down(): void
    {
        if (Schema::getConnection()->getDriverName() !== 'pgsql') {
            return;
        }

        Schema::dropIfExists('microscope_settings');
    }
};
