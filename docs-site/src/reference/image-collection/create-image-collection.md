# CREATE IMAGE COLLECTION

## Synopsis

    CREATE IMAGE COLLECTION module.name
        [EXPORT LEVEL 'Hidden' | 'Public']
        [COMMENT 'description']
        [(
            IMAGE 'image_name' FROM FILE 'path',
            ...
        )];

## Description

Creates a new image collection in the specified module. Image collections bundle images (icons, logos, graphics) within a module. Images can be loaded from the filesystem during creation, with the format detected automatically from the file extension.

## Parameters

**module.name**
: The qualified name of the collection (e.g., `MyModule.AppIcons`).

**EXPORT LEVEL**
: Controls visibility from other modules. `'Hidden'` (default) restricts access to the owning module. `'Public'` makes images available to other modules.

**COMMENT**
: Documentation text for the collection.

**IMAGE 'name' FROM FILE 'path'**
: Loads an image from a file on disk. The path is relative to the current working directory. Supported formats: PNG, SVG, GIF, JPEG, BMP, WebP.

## Examples

### Empty collection

```sql
CREATE IMAGE COLLECTION MyModule.AppIcons;
```

### Public collection with description

```sql
CREATE IMAGE COLLECTION MyModule.SharedIcons
    EXPORT LEVEL 'Public'
    COMMENT 'Shared icons for all modules';
```

### Collection with images

```sql
CREATE IMAGE COLLECTION MyModule.NavigationIcons (
    IMAGE 'home' FROM FILE 'assets/home.png',
    IMAGE 'settings' FROM FILE 'assets/settings.svg',
    IMAGE 'profile' FROM FILE 'assets/profile.png'
);
```

### All options combined

```sql
CREATE IMAGE COLLECTION MyModule.BrandAssets
    EXPORT LEVEL 'Public'
    COMMENT 'Company branding assets' (
    IMAGE 'logo-dark' FROM FILE 'assets/logo-dark.png',
    IMAGE 'logo-light' FROM FILE 'assets/logo-light.png'
);
```

## See Also

[DROP IMAGE COLLECTION](drop-image-collection.md), [SHOW / DESCRIBE IMAGE COLLECTION](show-describe-image-collection.md)
