# Database Schema & Model

Schema and Model that have :
1. Warehouse.
2. Team.

## Warehouse Schema
1. Legacy compatibility
    Because Warehouse Schema is exist in legacy system before. we must aware about the migration. in new migration this project, create rack **If Only** that table is not exist.
2. This is legacy golang struct that reflected the schema. for now, the field and schema already accomodate this system. No need to change.
    ```
    type Warehouse struct {
        ID uint `json:"id" gorm:"primarykey;autoIncrement:false"`

        Name        string  `json:"name"`
        IsFull      bool    `json:"is_full"`
        UseFixedFee bool    `json:"use_fixed_fee"`
        FeeFix      float64 `json:"basic_fee_fix"`
        FeePercent  float32 `json:"fee_percent"`
        MaxFee      float64 `json:"max_fee"`

        Desc    string `json:"desc"`
        Address string `json:"address"`

        OpenTime   *time.Time `json:"open_time"`
        CloseTime  *time.Time `json:"close_time"`
        CloseOrder *time.Time `json:"close_order"`

        IsClosed bool `json:"is_closed"`

        Created time.Time `json:"created"`
        Deleted bool      `json:"deleted" gorm:"index"`
    }
    
    ```
3. if on `./warehouse_models` doesn't have golang model for that definition, duplicate legacy and place at `./warehouse_models/warehouse.go`


## Team Schema
1. Legacy compatibility
    Because Team Schema is exist in legacy system before. we must aware about the migration. in new migration this project, create rack **If Only** that table is not exist.
2. This is legacy golang struct that reflected the schema. for now, the field and schema already accomodate this system. No need to change.
    ```
    type Team struct {
        ID                uint     `json:"id" gorm:"primarykey"`
        Type              TeamType `json:"type"`
        Name              string   `json:"name"`
        TeamCode          TeamCode `json:"team_code" gorm:"index:team_code_unique,unique" binding:"required"`
        Description       string   `json:"desc"`

        Deleted  bool         `json:"deleted"`
    }
    ```
3. if on `./warehouse_models` doesn't have golang model for that definition, duplicate legacy and place at `./warehouse_models/team.go`