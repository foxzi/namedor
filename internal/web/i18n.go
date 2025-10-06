package web

import "fmt"

var translations = map[string]map[string]string{
    "en": {
        // General
        "GeoDNS Admin": "GeoDNS Admin",
        "Logout": "Logout",
        "DNS Zones": "DNS Zones",
        "Templates": "Templates",
        "DNS Templates": "DNS Templates",
        "Query Logs": "Query Logs",
        "Loading...": "Loading...",
        "+ New Zone": "+ New Zone",
        "+ New Template": "+ New Template",
        "Query logs viewer coming soon...": "Query logs viewer coming soon...",
        "Cancel": "Cancel",
        "EN": "EN",
        "RU": "RU",

        // Login
        "Username": "Username",
        "Password": "Password",
        "Login": "Login",
        "Invalid username or password": "Invalid username or password",

        // Zones list
        "Zone Name": "Zone Name",
        "Records": "Records",
        "Actions": "Actions",
        "No zones found. Create your first zone!": "No zones found. Create your first zone!",
        "View Records": "View Records",
        "Delete": "Delete",
        "Delete zone %s?": "Delete zone %s?",

        // New zone form
        "Create New Zone": "Create New Zone",
        "Create": "Create",
        "Zone name is required": "Zone name is required",
        "Error creating zone: %s": "Error creating zone: %s",
        "Invalid zone ID": "Invalid zone ID",
        "Zone not found": "Zone not found",
        "Error loading records": "Error loading records",
        "Error loading templates": "Error loading templates",
        "Error loading zones": "Error loading zones",
        "Error deleting zone": "Error deleting zone",

        // Records
        "← Back to Zones": "← Back to Zones",
        "Records for %s": "Records for %s",
        "+ Add Record": "+ Add Record",
        "📋 Apply Template": "📋 Apply Template",
        "No records found. Add your first record!": "No records found. Add your first record!",
        "Name": "Name",
        "Type": "Type",
        "TTL": "TTL",
        "GeoIP": "GeoIP",
        "Data": "Data",
        "Edit": "Edit",
        "Delete this record?": "Delete this record?",
        "Add New Record": "Add New Record",
        "TTL (seconds)": "TTL (seconds)",
        "Data (IP/Value)": "Data (IP/Value)",
        "GeoIP Targeting (optional)": "GeoIP Targeting (optional)",
        "Country Code": "Country Code",
        "Continent Code": "Continent Code",
        "ASN": "ASN",
        "Subnet": "Subnet",
        "Add Record": "Add Record",
        "Name, type, and data are required": "Name, type, and data are required",
        "Error creating record set: %s": "Error creating record set: %s",
        "Error creating record: %s": "Error creating record: %s",
        "Error deleting record": "Error deleting record",

        // Templates
        "Template Name": "Template Name",
        "Description": "Description",
        "No templates found. Create your first template!": "No templates found. Create your first template!",
        "View": "View",
        "Delete template '%s'?": "Delete template '%s'?",
        "Create New Template": "Create New Template",
        "Create Template": "Create Template",
        "Template name is required": "Template name is required",
        "Error creating template: %s": "Error creating template: %s",
        "Brief description of this template": "Brief description of this template",
        "Invalid template ID": "Invalid template ID",
        "Template not found": "Template not found",
        "Template Records": "Template Records",
        "No records in this template.": "No records in this template.",
        "Country: %s": "Country: %s",
        "Continent: %s": "Continent: %s",
        "ASN: %d": "ASN: %d",
        "Subnet: %s": "Subnet: %s",
        "Edit Template: %s": "Edit Template: %s",
        "Update Template": "Update Template",
        "No records yet. Add records to this template.": "No records yet. Add records to this template.",
        "Add Record": "Add Record",
        "Error updating template: %s": "Error updating template: %s",
        "Error deleting template": "Error deleting template",
        "Add Template Record": "Add Template Record",
        "Use placeholders: <code>{domain}</code> for zone name, <code>{subdomain}</code> for custom names": "Use placeholders: <code>{domain}</code> for zone name, <code>{subdomain}</code> for custom names",
        "Name (supports placeholders)": "Name (supports placeholders)",
        "Data (supports placeholders)": "Data (supports placeholders)",
        "Apply Template": "Apply Template",
        "Zone: %s": "Zone: %s",
        "This will create %d records:": "This will create %d records:",
    },
    "ru": {
        // General
        "GeoDNS Admin": "GeoDNS Админ",
        "Logout": "Выход",
        "DNS Zones": "DNS Зоны",
        "Templates": "Шаблоны",
        "DNS Templates": "DNS Шаблоны",
        "Query Logs": "Логи запросов",
        "Loading...": "Загрузка...",
        "+ New Zone": "+ Новая зона",
        "+ New Template": "+ Новый шаблон",
        "Query logs viewer coming soon...": "Просмотр логов скоро появится...",
        "Cancel": "Отмена",
        "EN": "EN",
        "RU": "RU",

        // Login
        "Username": "Логин",
        "Password": "Пароль",
        "Login": "Войти",
        "Invalid username or password": "Неверные логин или пароль",

        // Zones list
        "Zone Name": "Имя зоны",
        "Records": "Записей",
        "Actions": "Действия",
        "No zones found. Create your first zone!": "Зон нет. Создайте первую зону!",
        "View Records": "Просмотр записей",
        "Delete": "Удалить",
        "Delete zone %s?": "Удалить зону %s?",

        // New zone form
        "Create New Zone": "Создать новую зону",
        "Create": "Создать",
        "Zone name is required": "Требуется имя зоны",
        "Error creating zone: %s": "Ошибка создания зоны: %s",
        "Invalid zone ID": "Некорректный ID зоны",
        "Zone not found": "Зона не найдена",
        "Error loading records": "Ошибка загрузки записей",
        "Error loading templates": "Ошибка загрузки шаблонов",
        "Error loading zones": "Ошибка загрузки зон",
        "Error deleting zone": "Ошибка удаления зоны",

        // Records
        "← Back to Zones": "← Назад к зонам",
        "Records for %s": "Записи для %s",
        "+ Add Record": "+ Добавить запись",
        "📋 Apply Template": "📋 Применить шаблон",
        "No records found. Add your first record!": "Записей нет. Добавьте первую запись!",
        "Name": "Имя",
        "Type": "Тип",
        "TTL": "TTL",
        "GeoIP": "GeoIP",
        "Data": "Данные",
        "Edit": "Изменить",
        "Delete this record?": "Удалить эту запись?",
        "Add New Record": "Добавить запись",
        "TTL (seconds)": "TTL (сек)",
        "Data (IP/Value)": "Данные (IP/значение)",
        "GeoIP Targeting (optional)": "GeoIP-таргетинг (опционально)",
        "Country Code": "Код страны",
        "Continent Code": "Код континента",
        "ASN": "ASN",
        "Subnet": "Подсеть",
        "Add Record": "Добавить",
        "Name, type, and data are required": "Имя, тип и данные обязательны",
        "Error creating record set: %s": "Ошибка создания набора записей: %s",
        "Error creating record: %s": "Ошибка создания записи: %s",
        "Error deleting record": "Ошибка удаления записи",

        // Templates
        "Template Name": "Имя шаблона",
        "Description": "Описание",
        "No templates found. Create your first template!": "Шаблонов нет. Создайте первый!",
        "View": "Просмотр",
        "Delete template '%s'?": "Удалить шаблон '%s'?",
        "Create New Template": "Создать новый шаблон",
        "Create Template": "Создать шаблон",
        "Template name is required": "Требуется имя шаблона",
        "Error creating template: %s": "Ошибка создания шаблона: %s",
        "Brief description of this template": "Краткое описание шаблона",
        "Invalid template ID": "Некорректный ID шаблона",
        "Template not found": "Шаблон не найден",
        "Template Records": "Записи шаблона",
        "No records in this template.": "В этом шаблоне нет записей.",
        "Country: %s": "Страна: %s",
        "Continent: %s": "Континент: %s",
        "ASN: %d": "ASN: %d",
        "Subnet: %s": "Подсеть: %s",
        "Edit Template: %s": "Редактировать шаблон: %s",
        "Update Template": "Обновить шаблон",
        "No records yet. Add records to this template.": "Записей пока нет. Добавьте записи.",
        "Add Record": "Добавить запись",
        "Error updating template: %s": "Ошибка обновления шаблона: %s",
        "Error deleting template": "Ошибка удаления шаблона",
        "Add Template Record": "Добавить запись шаблона",
        "Use placeholders: <code>{domain}</code> for zone name, <code>{subdomain}</code> for custom names": "Используйте плейсхолдеры: <code>{domain}</code> для имени зоны, <code>{subdomain}</code> для пользовательских имён",
        "Name (supports placeholders)": "Имя (поддерживает плейсхолдеры)",
        "Data (supports placeholders)": "Данные (поддерживают плейсхолдеры)",
        "Apply Template": "Применить шаблон",
        "Zone: %s": "Зона: %s",
        "This will create %d records:": "Будет создано %d записей:",
    },
}

func tr(lang, key string) string {
    if m, ok := translations[lang]; ok {
        if v, ok2 := m[key]; ok2 {
            return v
        }
    }
    // fallback to en
    if m, ok := translations["en"]; ok {
        if v, ok2 := m[key]; ok2 {
            return v
        }
    }
    return key
}

func trf(lang, key string, a ...any) string {
    return fmt.Sprintf(tr(lang, key), a...)
}
        // Record edit
        "Invalid record ID": "Invalid record ID",
        "Record not found": "Record not found",
        "RRSet not found": "RRSet not found",
        "Edit Record": "Edit Record",
        "Name cannot be changed": "Name cannot be changed",
        "Type cannot be changed": "Type cannot be changed",
        "Update Record": "Update Record",
        "Data is required": "Data is required",
        "Error updating record: %s": "Error updating record: %s",
        "Error updating TTL: %s": "Error updating TTL: %s",
        // Record edit
        "Invalid record ID": "Некорректный ID записи",
        "Record not found": "Запись не найдена",
        "RRSet not found": "Набор записей (RRSet) не найден",
        "Edit Record": "Изменить запись",
        "Name cannot be changed": "Имя нельзя изменить",
        "Type cannot be changed": "Тип нельзя изменить",
        "Update Record": "Обновить запись",
        "Data is required": "Требуются данные",
        "Error updating record: %s": "Ошибка обновления записи: %s",
        "Error updating TTL: %s": "Ошибка обновления TTL: %s",
