IF DB_ID(N'db_tui') IS NULL
BEGIN
    CREATE DATABASE [db_tui];
END;
GO

USE [db_tui];
GO

IF OBJECT_ID(N'dbo.countries', N'U') IS NULL
BEGIN
    CREATE TABLE dbo.countries (
        country_code CHAR(2) NOT NULL PRIMARY KEY,
        name NVARCHAR(100) NOT NULL
    );
END;
GO

IF OBJECT_ID(N'dbo.cities', N'U') IS NULL
BEGIN
    CREATE TABLE dbo.cities (
        city_id INT NOT NULL PRIMARY KEY,
        country_code CHAR(2) NOT NULL,
        name NVARCHAR(100) NOT NULL,
        population INT NULL,
        CONSTRAINT FK_cities_countries
            FOREIGN KEY (country_code) REFERENCES dbo.countries (country_code)
    );
END;
GO

IF NOT EXISTS (SELECT 1 FROM dbo.countries WHERE country_code = 'AR')
    INSERT INTO dbo.countries (country_code, name) VALUES ('AR', N'Argentina');
IF NOT EXISTS (SELECT 1 FROM dbo.countries WHERE country_code = 'JP')
    INSERT INTO dbo.countries (country_code, name) VALUES ('JP', N'Japan');
IF NOT EXISTS (SELECT 1 FROM dbo.countries WHERE country_code = 'US')
    INSERT INTO dbo.countries (country_code, name) VALUES ('US', N'United States');
GO

IF NOT EXISTS (SELECT 1 FROM dbo.cities WHERE city_id = 1)
    INSERT INTO dbo.cities (city_id, country_code, name, population)
    VALUES (1, 'AR', N'Buenos Aires', 3120612);
IF NOT EXISTS (SELECT 1 FROM dbo.cities WHERE city_id = 2)
    INSERT INTO dbo.cities (city_id, country_code, name, population)
    VALUES (2, 'JP', N'Tokyo', 13960000);
IF NOT EXISTS (SELECT 1 FROM dbo.cities WHERE city_id = 3)
    INSERT INTO dbo.cities (city_id, country_code, name, population)
    VALUES (3, 'US', N'New York', 8258035);
GO

CREATE OR ALTER VIEW dbo.city_directory
AS
SELECT
    city.city_id,
    city.name AS city_name,
    country.country_code,
    country.name AS country_name,
    city.population
FROM dbo.cities AS city
JOIN dbo.countries AS country
    ON country.country_code = city.country_code;
GO

SET ANSI_NULLS ON;
SET ANSI_PADDING ON;
SET ANSI_WARNINGS ON;
SET ARITHABORT ON;
SET CONCAT_NULL_YIELDS_NULL ON;
SET QUOTED_IDENTIFIER ON;
SET NUMERIC_ROUNDABORT OFF;
GO

CREATE OR ALTER VIEW dbo.country_city_counts
WITH SCHEMABINDING
AS
SELECT
    city.country_code,
    COUNT_BIG(*) AS city_count
FROM dbo.cities AS city
GROUP BY city.country_code;
GO

IF NOT EXISTS (
    SELECT 1
    FROM sys.indexes
    WHERE object_id = OBJECT_ID(N'dbo.country_city_counts')
        AND index_id = 1
)
BEGIN
    CREATE UNIQUE CLUSTERED INDEX CIX_country_city_counts
        ON dbo.country_city_counts (country_code);
END;
GO

-- A schema whose only object is an indexed view. Schema object discovery must
-- still report a "views" group for it, otherwise the view is unreachable from
-- the database explorer.
IF SCHEMA_ID(N'reporting') IS NULL
BEGIN
    EXEC(N'CREATE SCHEMA reporting');
END;
GO

CREATE OR ALTER VIEW reporting.city_totals
WITH SCHEMABINDING
AS
SELECT
    city.country_code,
    COUNT_BIG(*) AS city_count
FROM dbo.cities AS city
GROUP BY city.country_code;
GO

IF NOT EXISTS (
    SELECT 1
    FROM sys.indexes
    WHERE object_id = OBJECT_ID(N'reporting.city_totals')
        AND index_id = 1
)
BEGIN
    CREATE UNIQUE CLUSTERED INDEX CIX_city_totals
        ON reporting.city_totals (country_code);
END;
GO

CREATE OR ALTER FUNCTION dbo.city_count()
RETURNS INT
AS
BEGIN
    RETURN (SELECT COUNT(*) FROM dbo.cities);
END;
GO

CREATE OR ALTER FUNCTION dbo.cities_by_country(@country_code CHAR(2))
RETURNS TABLE
AS
RETURN (
    SELECT city_id, country_code, name, population
    FROM dbo.cities
    WHERE country_code = @country_code
);
GO
