<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        if (Schema::getConnection()->getDriverName() !== 'pgsql') {
            return;
        }

        if (Schema::hasTable('microscope_options')) {
            return;
        }

        Schema::create('microscope_options', function (Blueprint $table) {
            $table->string('key')->primary();
            $table->jsonb('value');
            $table->timestampTz('updated_at')->useCurrent();
        });

        DB::table('microscope_options')->insertOrIgnore([
            'key' => 'redact_sensitive',
            'value' => json_encode(false),
            'updated_at' => now(),
        ]);
    }

    public function down(): void
    {
        if (Schema::getConnection()->getDriverName() !== 'pgsql') {
            return;
        }

        Schema::dropIfExists('microscope_options');
    }
};
