-- MySQL dump 10.13  Distrib 8.4.5, for Win64 (x86_64)
--
-- Host: localhost    Database: file-store
-- ------------------------------------------------------
-- Server version	8.4.5

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!50503 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

--
-- Table structure for table `favorite`
--

DROP TABLE IF EXISTS `favorite`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `favorite` (
  `id` int NOT NULL AUTO_INCREMENT,
  `user_id` int NOT NULL COMMENT '收藏者ID（文件的拥有者）',
  `file_id` varchar(40) COLLATE utf8mb4_general_ci NOT NULL COMMENT '文件ID',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_fav` (`user_id`,`file_id`),
  KEY `favorite_ibfk_2` (`file_id`),
  CONSTRAINT `favorite_ibfk_1` FOREIGN KEY (`user_id`) REFERENCES `user` (`id`),
  CONSTRAINT `favorite_ibfk_2` FOREIGN KEY (`file_id`) REFERENCES `file` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=18 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `favorite`
--

LOCK TABLES `favorite` WRITE;
/*!40000 ALTER TABLE `favorite` DISABLE KEYS */;
INSERT INTO `favorite` VALUES (1,8,'f0bc2089-5c23-45fe-91fd-f6681137ae52','2025-08-16 07:41:02'),(2,8,'ce03f3ea-d62d-48a1-867e-9b2a6c0bba0b','2025-08-16 07:54:33'),(3,8,'16b064f5-5b2c-4274-87c6-5ffe112130d9','2025-08-16 08:20:46'),(6,8,'c6cff8b5-64b6-451a-8b47-4631850fc105','2025-08-17 15:12:29'),(7,18,'e9d71e76-4a51-42a8-b6ce-c5ccce8ee992','2025-08-19 07:37:48'),(15,20,'f946149e-c834-4582-8cb8-90d3f451fa44','2025-12-15 11:59:35'),(16,20,'c7ce1eec-bd08-44df-8f09-a95072b09778','2025-12-20 07:06:08');
/*!40000 ALTER TABLE `favorite` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `file`
--

DROP TABLE IF EXISTS `file`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `file` (
  `id` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'UUID或OSS标识',
  `user_id` int NOT NULL COMMENT '用户ID',
  `name` varchar(255) COLLATE utf8mb4_general_ci NOT NULL COMMENT '原始文件名',
  `size` bigint(20) unsigned zerofill DEFAULT NULL COMMENT '字节大小',
  `size_str` varchar(20) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '文件可读大小，如2.8MB',
  `is_dir` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否为目录',
  `file_extension` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '文件扩展名',
  `file_url` varchar(255) COLLATE utf8mb4_general_ci NOT NULL COMMENT '文件访问URL',
  `thumbnail_url` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '缩略图URL',
  `oss_object_key` varchar(1024) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT 'OSS对象键',
  `file_hash` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT 'SHA256',
  `parent_id` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `is_deleted` tinyint(1) DEFAULT '0' COMMENT '软删除标志',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_user_parent_name` (`user_id`,`parent_id`,`name`),
  KEY `file_ibfk_2` (`parent_id`),
  CONSTRAINT `file_ibfk_1` FOREIGN KEY (`user_id`) REFERENCES `user` (`id`),
  CONSTRAINT `file_ibfk_2` FOREIGN KEY (`parent_id`) REFERENCES `file` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `file`
--

LOCK TABLES `file` WRITE;
/*!40000 ALTER TABLE `file` DISABLE KEYS */;
INSERT INTO `file` VALUES ('0b9bb789-20d1-4f8e-b8dd-d4e0ed48d5b9',19,'黄埔班-185-5-1-伏仰-基于深度学习的Twitter社交机器人检测 .pptx',00000000000003346317,'3.19 MB',0,'pptx','https://go-cloud-store.oss-cn-shenzhen.aliyuncs.com/files/19/2cdd2c53-67ca-40b0-af02-409d18634efb.pptx','https://go-cloud-store.oss-cn-shenzhen.aliyuncs.com/files/19/2cdd2c53-67ca-40b0-af02-409d18634efb.pptx','files/19/2cdd2c53-67ca-40b0-af02-409d18634efb.pptx','969d2c3549fca26bb344ce75a118a8aa','19334b9d-f5c6-48a9-837a-d80db073858b',0,'2025-08-27 14:28:23','2025-08-27 14:28:23'),('0d320bd8-c751-45c7-a5b5-454c7ea17eb4',8,'b.jpg',00000000000000275798,'269.33 KB',0,'jpg','https://go-cloud-storage.oss-cn-beijing.aliyuncs.com/files/8/76a169e8-72b0-4d34-b809-16291d1ecd2a/0d320bd8-c751-45c7-a5b5-454c7ea17eb4.jpg','https://go-cloud-storage.oss-cn-beijing.aliyuncs.com/files/8/76a169e8-72b0-4d34-b809-16291d1ecd2a/0d320bd8-c751-45c7-a5b5-454c7ea17eb4.jpg','files/8/76a169e8-72b0-4d34-b809-16291d1ecd2a/0d320bd8-c751-45c7-a5b5-454c7ea17eb4.jpg','3bca133d0cc88a5416f706ecfedd7617a7fb1a7a1387fed57cd876ced3042c9c','76a169e8-72b0-4d34-b809-16291d1ecd2a',0,'2025-08-14 09:58:53','2025-08-17 11:50:01'),('1129f732-f060-452e-a070-3f8aaaf212e1',19,'我的资源',00000000000000000000,'-',1,'','','','','','19334b9d-f5c6-48a9-837a-d80db073858b',0,'2025-08-21 13:10:51','2025-08-21 13:10:50'),('16b064f5-5b2c-4274-87c6-5ffe112130d9',8,'235959-1655135999aab8.jpg',00000000000000511741,'499.75 KB',0,'jpg','https://go-cloud-storage.oss-cn-beijing.aliyuncs.com/files/8/d9749181-f32c-44a1-b7bd-8fd0d7f2dbc2/16b064f5-5b2c-4274-87c6-5ffe112130d9.jpg','https://go-cloud-storage.oss-cn-beijing.aliyuncs.com/files/8/d9749181-f32c-44a1-b7bd-8fd0d7f2dbc2/16b064f5-5b2c-4274-87c6-5ffe112130d9.jpg','files/8/d9749181-f32c-44a1-b7bd-8fd0d7f2dbc2/16b064f5-5b2c-4274-87c6-5ffe112130d9.jpg','e12ac388a94305281462e851c987f38cb7f82e992c9ec9e60688d4145f751426','d9749181-f32c-44a1-b7bd-8fd0d7f2dbc2',0,'2025-08-16 08:20:35','2025-08-16 08:20:35'),('16c589dc-5d1f-464c-9c97-d442a9291d7d',8,'测试ABC',00000000000000000000,'-',1,'','',NULL,'','','76a169e8-72b0-4d34-b809-16291d1ecd2a',0,'2025-08-05 11:34:59','2025-08-14 09:35:42'),('17aad23a-a5ee-4cfa-972e-0bf2461f371d',19,'无人之岛.mp4',00000000000101772076,'97.06 MB',0,'mp4','https://go-cloud-store.oss-cn-shenzhen.aliyuncs.com/files/19/53d07fe7-e7c8-48a6-9eac-42b789396d2d.mp4','https://go-cloud-store.oss-cn-shenzhen.aliyuncs.com/files/19/53d07fe7-e7c8-48a6-9eac-42b789396d2d.mp4','files/19/53d07fe7-e7c8-48a6-9eac-42b789396d2d.mp4','34b82a728c4ff8f808f20bf644a6c4a3','19334b9d-f5c6-48a9-837a-d80db073858b',0,'2025-08-23 04:28:58','2025-08-23 04:28:58'),('18c55cfd-7def-4c55-abe3-2844e603d075',20,'2402.06196v3.pdf',00000000000004882035,'4.66 MB',0,'pdf','http://192.168.101.129:9001/go-cloud-storage/files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/987d94f7-9f81-42e7-b560-af87402bfa1e.pdf','http://192.168.101.129:9001/go-cloud-storage/files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/987d94f7-9f81-42e7-b560-af87402bfa1e.pdf','files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/987d94f7-9f81-42e7-b560-af87402bfa1e.pdf','c5b884dc070cc6ed3e4ec06214fa84d4afff0873e7e7f66f8dcdcd2285e10914','8983662f-86d1-41ab-8aad-a9c1c21d03dc',0,'2026-01-18 03:36:35','2026-01-18 03:36:35'),('19334b9d-f5c6-48a9-837a-d80db073858b',19,'/',00000000000000000000,'',1,'','','','','',NULL,0,'2025-08-21 12:58:36','2025-08-21 12:58:35'),('199dce84-ae65-47e8-a0bf-46cbdb7273ea',19,'glove.twitter.27B.25d.txt',00000000000257699726,'245.76 MB',0,'txt','https://go-cloud-store.oss-cn-shenzhen.aliyuncs.com/files/19/6285f67f-4b77-44b9-ad14-82ca4b9ac9e5.txt','https://go-cloud-store.oss-cn-shenzhen.aliyuncs.com/files/19/6285f67f-4b77-44b9-ad14-82ca4b9ac9e5.txt','files/19/6285f67f-4b77-44b9-ad14-82ca4b9ac9e5.txt','f38598c6654cba5e6d0cef9bb833bdb1','19334b9d-f5c6-48a9-837a-d80db073858b',0,'2025-08-27 14:19:47','2025-08-27 14:19:47'),('1b155896-cfd6-4da8-a356-44612e3c6724',20,'无人之岛.mp4',00000000000101772076,'97.06 MB',0,'mp4','http://192.168.101.129:9001/go-cloud-storage/files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/017bda08-3911-4b14-86ec-2ee73ac8ec70.mp4','http://192.168.101.129:9001/go-cloud-storage/files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/017bda08-3911-4b14-86ec-2ee73ac8ec70_thumb.jpg','files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/017bda08-3911-4b14-86ec-2ee73ac8ec70.mp4','74116b7f6df471e14f1fe44a9a6b6b43d7c98246930293f4bdf6404690a035e9','8983662f-86d1-41ab-8aad-a9c1c21d03dc',0,'2025-12-10 13:57:23','2025-12-10 13:57:23'),('20c4a12d-f8b4-456a-a4f6-6b5435d02612',20,'G.E.M.邓紫棋 - 多远都要在一起.ncm',00000000000009063594,'8.64 MB',0,'ncm','http://192.168.101.129:9001/go-cloud-storage/files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/14fca37a-39d9-4b71-87ad-a703560411b2.ncm','http://192.168.101.129:9001/go-cloud-storage/files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/14fca37a-39d9-4b71-87ad-a703560411b2.ncm','files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/14fca37a-39d9-4b71-87ad-a703560411b2.ncm','86f2687d56e142185af28017b05a94c8635293669f8a48636c9b585f07f1d1b5','8983662f-86d1-41ab-8aad-a9c1c21d03dc',0,'2025-12-20 06:51:54','2025-12-20 06:51:54'),('21289a97-04dc-4aa4-a6d7-181b7e319a2e',20,'77_Who_Routes_the_Router_Rethi.pdf',00000000000002296151,'2.19 MB',0,'pdf','http://192.168.101.129:9001/go-cloud-storage/files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/d42c725f-c239-4adc-86d9-9e7c9ad7b552.pdf','http://192.168.101.129:9001/go-cloud-storage/files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/d42c725f-c239-4adc-86d9-9e7c9ad7b552.pdf','files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/d42c725f-c239-4adc-86d9-9e7c9ad7b552.pdf','0b55702bbfc1fd81b429050fc7457b23f22672adea532fb8078fc47f300ae1b6','8983662f-86d1-41ab-8aad-a9c1c21d03dc',0,'2026-01-18 03:39:48','2026-01-18 03:39:48'),('224c2b4e-0d0b-4af7-997f-1c76cd6a2527',20,'AA',00000000000000000000,'-',1,'','','','','','8983662f-86d1-41ab-8aad-a9c1c21d03dc',0,'2025-11-25 08:32:10','2025-11-25 08:32:09'),('25298edf-a96d-42bb-bde4-d7140bd94828',11,'9cd8d4e74ce0b78253dc4dcc16995cb5.jpg',00000000000000978724,'955.79 KB',0,'jpg','https://go-cloud-storage.oss-cn-beijing.aliyuncs.com/files/11/7fdbe4ef-d387-49d6-a065-682ba7ae5abc/25298edf-a96d-42bb-bde4-d7140bd94828.jpg','https://go-cloud-storage.oss-cn-beijing.aliyuncs.com/files/11/7fdbe4ef-d387-49d6-a065-682ba7ae5abc/25298edf-a96d-42bb-bde4-d7140bd94828.jpg','files/11/7fdbe4ef-d387-49d6-a065-682ba7ae5abc/25298edf-a96d-42bb-bde4-d7140bd94828.jpg','2b3385671d5883b47489d19815f44b8f871af423e443d87bcd8bca4102fc4098','7fdbe4ef-d387-49d6-a065-682ba7ae5abc',0,'2025-08-18 14:15:01','2025-08-18 14:15:01'),('2893f366-b976-4c4e-b8cf-e41b9df7a190',20,'77c6a7efce1b9d1601c3a91ca9aa86868d5464c0.jpeg',00000000000000035407,'34.58 KB',0,'jpeg','http://192.168.101.129:9001/go-cloud-storage/files/20/224c2b4e-0d0b-4af7-997f-1c76cd6a2527/c2c0f16c-5b0a-432c-a915-b46eda51fa3d.jpeg','http://192.168.101.129:9001/go-cloud-storage/files/20/224c2b4e-0d0b-4af7-997f-1c76cd6a2527/c2c0f16c-5b0a-432c-a915-b46eda51fa3d_thumb.jpg','files/20/224c2b4e-0d0b-4af7-997f-1c76cd6a2527/c2c0f16c-5b0a-432c-a915-b46eda51fa3d.jpeg','d81891f6bfaba7f28b7093a2f5277cf6b7158f47f1770203124d57c1919b4945','224c2b4e-0d0b-4af7-997f-1c76cd6a2527',0,'2026-01-18 05:37:52','2026-01-18 05:37:52'),('2953cf23-fca9-4e1e-a826-8f316ec2c684',19,'方班-173-8-2-高海圳-SynPrompt基于方面情感分析的语法感知增强提示词工程.pptx',00000000000025942187,'24.74 MB',0,'pptx','https://go-cloud-store.oss-cn-shenzhen.aliyuncs.com/files/19/19334b9d-f5c6-48a9-837a-d80db073858b/2953cf23-fca9-4e1e-a826-8f316ec2c684.pptx','https://go-cloud-store.oss-cn-shenzhen.aliyuncs.com/files/19/19334b9d-f5c6-48a9-837a-d80db073858b/2953cf23-fca9-4e1e-a826-8f316ec2c684.pptx','files/19/19334b9d-f5c6-48a9-837a-d80db073858b/2953cf23-fca9-4e1e-a826-8f316ec2c684.pptx','4d5d9593e12bfef2735d704a98303e944b18a767c91c3b88550c18d344eb15fb','19334b9d-f5c6-48a9-837a-d80db073858b',0,'2025-08-24 08:40:54','2025-08-24 08:40:54'),('29550fb0-bf51-45a3-b0e9-f69b5ffb0547',19,'HSK test.docx',00000000000000082217,'80.29 KB',0,'docx','https://go-cloud-store.oss-cn-shenzhen.aliyuncs.com/files/19/27849fd4-d8b4-471d-b998-89b233eddb6b.docx','https://go-cloud-store.oss-cn-shenzhen.aliyuncs.com/files/19/27849fd4-d8b4-471d-b998-89b233eddb6b.docx','files/19/27849fd4-d8b4-471d-b998-89b233eddb6b.docx','810f7dbc1d2ea086014dcef8b4f1e095','19334b9d-f5c6-48a9-837a-d80db073858b',0,'2025-08-24 08:39:51','2025-08-24 08:39:51'),('3396c401-de55-43f0-bbb9-2ea53386a23e',19,'BigmodelPoster.png',00000000000000104268,'101.82 KB',0,'png','https://go-cloud-store.oss-cn-shenzhen.aliyuncs.com/files/19/8bfbdc98-26b8-41e9-8690-872348e88993.png','https://go-cloud-store.oss-cn-shenzhen.aliyuncs.com/files/19/8bfbdc98-26b8-41e9-8690-872348e88993.png','files/19/8bfbdc98-26b8-41e9-8690-872348e88993.png','cca43c4b4adff2f8478a48b8214b64c4',NULL,0,'2025-09-05 13:30:43','2025-09-05 13:30:43'),('34b82a728c4ff8f808f20bf644a6c4a3',18,'无人之岛.mp4',00000000000101772076,'97.06 MB',0,'mp4','https://go-cloud-storage.oss-cn-beijing.aliyuncs.com/files/18/34b82a728c4ff8f808f20bf644a6c4a3.mp4','https://go-cloud-storage.oss-cn-beijing.aliyuncs.com/files/18/34b82a728c4ff8f808f20bf644a6c4a3.mp4','files/18/34b82a728c4ff8f808f20bf644a6c4a3.mp4','34b82a728c4ff8f808f20bf644a6c4a3','72915093-55c7-4ce3-a8fd-45830883ec53',0,'2025-08-21 07:51:36','2025-08-21 07:51:36'),('3fe885dc-6927-4147-a3c7-5f18b12e5edc',19,'A目录',00000000000000000000,'-',1,'','','','','','19334b9d-f5c6-48a9-837a-d80db073858b',0,'2025-08-21 13:48:17','2025-08-21 13:48:16'),('4ac9e908-13c9-4b6e-81c7-f6eaee765fda',20,'模糊的边界_高校研究生何以既无研究也无生活.pdf',00000000000004017693,'3.83 MB',0,'pdf','http://192.168.101.129:9001/go-cloud-storage/files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/3156d7a8-9d59-4933-8319-bed8f2e8d667.pdf','http://192.168.101.129:9001/go-cloud-storage/files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/3156d7a8-9d59-4933-8319-bed8f2e8d667.pdf','files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/3156d7a8-9d59-4933-8319-bed8f2e8d667.pdf','4d185fe07082cca81fba0d16c847491e53cd904badc5d23f00bc539e78e9af75','8983662f-86d1-41ab-8aad-a9c1c21d03dc',0,'2026-01-18 02:19:56','2026-01-18 02:19:56'),('50bc9ed0-8b0f-4268-a51d-fb4fed13691f',19,'音乐',00000000000000000000,'-',1,'','','','','','1129f732-f060-452e-a070-3f8aaaf212e1',0,'2025-08-21 15:00:05','2025-08-21 15:00:05'),('56066a56-6271-40d9-bd89-c236f65d6753',8,'Deep_Learning_Based_Social_Bot_Detection_on_Twitter.pdf',00000000000005279344,'5.03 MB',0,'pdf','',NULL,'files/8/76a169e8-72b0-4d34-b809-16291d1ecd2a/56066a56-6271-40d9-bd89-c236f65d6753.pdf','8b5d36b2e4245e51c62883961c189f4306d909b7961e1250b32e674f2a6ddf84','76a169e8-72b0-4d34-b809-16291d1ecd2a',1,'2025-08-08 10:54:52','2025-08-14 09:31:35'),('5969825e-e8be-47c3-ad9f-055e27e4787e',20,'93e01746e49ebd2d53c6bc2d26e7c466.jpg',00000000000001230141,'1.17 MB',0,'jpg','http://192.168.101.129:9001/go-cloud-storage/files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/819ba84f-ac12-4808-b77c-b8765eb489b1.jpg','http://192.168.101.129:9001/go-cloud-storage/files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/819ba84f-ac12-4808-b77c-b8765eb489b1_thumb.jpg','files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/819ba84f-ac12-4808-b77c-b8765eb489b1.jpg','c63b09a12e8115329ef50688ddba1fb791239bc02efc0cace4aab1c1b7c98bc8','8983662f-86d1-41ab-8aad-a9c1c21d03dc',0,'2025-12-07 08:16:44','2025-12-07 08:16:44'),('61363c5c-2606-4a66-b0bf-713ae4799593',8,'测试文件夹',00000000000000000000,'-',1,'','',NULL,'','','76a169e8-72b0-4d34-b809-16291d1ecd2a',0,'2025-08-02 03:56:33','2025-08-14 09:35:51'),('626257ec-4857-4c00-9708-177282a7a77c',8,'Linux操作系统面试题.pdf',00000000000001317487,'1.25 MB',0,'pdf','https://go-cloud-storage.oss-cn-beijing.aliyuncs.com/files/8/56066a56-6271-40d9-bd89-c236f65d6753.pdf','https://go-cloud-storage.oss-cn-beijing.aliyuncs.com/files/8/56066a56-6271-40d9-bd89-c236f65d6753.pdf','files/8/76a169e8-72b0-4d34-b809-16291d1ecd2a/626257ec-4857-4c00-9708-177282a7a77c.pdf','0dc269647b357f4873fe443606e89353bd15d044c99b45c441a1e45fdd819e61','76a169e8-72b0-4d34-b809-16291d1ecd2a',0,'2025-08-08 08:08:47','2025-08-14 09:26:27'),('680117fa-388a-4a83-8870-91b98ff2c069',20,'Chain-of-Thought Prompting Elicits Reasoning in Large Language Models.pdf',00000000000000682765,'666.76 KB',0,'pdf','http://192.168.101.129:9001/go-cloud-storage/files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/3cf13f0a-eb3e-4675-b998-82cab3292cd6.pdf','http://192.168.101.129:9001/go-cloud-storage/files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/3cf13f0a-eb3e-4675-b998-82cab3292cd6.pdf','files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/3cf13f0a-eb3e-4675-b998-82cab3292cd6.pdf','d270421a82001ae42e8bda69c3b6a5a4f7ba64084e9e134c245cc9a1d7b84371','8983662f-86d1-41ab-8aad-a9c1c21d03dc',0,'2026-01-18 03:18:34','2026-01-18 03:18:34'),('6c668186-3ccd-4efe-bc19-ca6012b88d07',19,'摇滚',00000000000000000000,'-',1,'','','','','','50bc9ed0-8b0f-4268-a51d-fb4fed13691f',0,'2025-08-22 11:50:23','2025-08-22 11:50:22'),('6cd0c92f-9ffe-4e25-a3e3-a6d231ecd2fb',19,'示范班-124-4-2-华南理工大学-李亚洲-LLM-Fuzzer大型语言模型越狱的扩展评估.pptx',00000000000001764201,'1.68 MB',0,'pptx','https://go-cloud-store.oss-cn-shenzhen.aliyuncs.com/files/19/7bfcec9c-6570-40df-935f-61f438112d56.pptx','https://go-cloud-store.oss-cn-shenzhen.aliyuncs.com/files/19/7bfcec9c-6570-40df-935f-61f438112d56.pptx','files/19/7bfcec9c-6570-40df-935f-61f438112d56.pptx','cd731a2108b4ef974ed2cb10978d8655','19334b9d-f5c6-48a9-837a-d80db073858b',1,'2025-08-27 14:52:43','2025-11-22 13:31:06'),('72915093-55c7-4ce3-a8fd-45830883ec53',18,'/',00000000000000000000,'',1,'','','','','',NULL,0,'2025-08-19 02:34:19','2025-08-19 02:34:18'),('74f5d474-86f1-4123-83e0-8f379cdf8623',19,'PLeak-针对大型语言模型应用程序的即时泄漏攻击.pdf',00000000000002635664,'2.51 MB',0,'pdf','https://go-cloud-store.oss-cn-shenzhen.aliyuncs.com/files/19/072540bb-6ef3-4f5f-93d9-240ca06ce66c.pdf','https://go-cloud-store.oss-cn-shenzhen.aliyuncs.com/files/19/072540bb-6ef3-4f5f-93d9-240ca06ce66c.pdf','files/19/072540bb-6ef3-4f5f-93d9-240ca06ce66c.pdf','7e577b11851d108de24edd09580ddbf7','19334b9d-f5c6-48a9-837a-d80db073858b',0,'2025-08-27 13:37:43','2025-08-27 13:37:43'),('76a169e8-72b0-4d34-b809-16291d1ecd2a',8,'/',00000000000000000000,'',1,'','',NULL,'','',NULL,0,'2025-08-02 02:37:24','2025-08-02 02:37:24'),('78c1b61e-2dc2-4cbd-a3d7-71678741719a',20,'BB',00000000000000000000,'-',1,'','','','','','224c2b4e-0d0b-4af7-997f-1c76cd6a2527',0,'2025-11-25 08:32:17','2025-11-29 14:26:09'),('7d02d637-cbc8-4b48-a9d9-a4eeca8504cc',20,'《中华人民共和国网络安全法》.docx',00000000000000050544,'49.36 KB',0,'docx','http://192.168.101.129:9001/go-cloud-storage/files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/940c9162-35ce-4874-9d3d-035a9834410f.docx','http://192.168.101.129:9001/go-cloud-storage/files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/940c9162-35ce-4874-9d3d-035a9834410f.docx','files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/940c9162-35ce-4874-9d3d-035a9834410f.docx','1db06ada14b7390628489a54474b3d71364497440359a647528efe38ba0d4100','8983662f-86d1-41ab-8aad-a9c1c21d03dc',0,'2026-01-18 02:31:34','2026-01-18 02:31:34'),('7fdbe4ef-d387-49d6-a065-682ba7ae5abc',11,'/',00000000000000000000,'',1,'','','','','',NULL,0,'2025-08-18 13:44:37','2025-08-18 13:44:37'),('82bb31dc-e48c-4c6b-837b-1db26f6286b7',16,'/',00000000000000000000,'',1,'','','','','',NULL,0,'2025-08-19 02:25:25','2025-08-19 02:25:25'),('86638497-cc7d-400b-80ff-4f90a95c7788',20,'Is It Just Me.mp4',00000000000037230272,'35.51 MB',0,'mp4','http://192.168.101.129:9001/go-cloud-storage/files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/d043db16-8cfe-47ee-933c-504bdbd76f71.mp4','http://192.168.101.129:9001/go-cloud-storage/files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/d043db16-8cfe-47ee-933c-504bdbd76f71_thumb.jpg','files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/d043db16-8cfe-47ee-933c-504bdbd76f71.mp4','b24a0f8d78ab4edf0e872633953e8b987f570d0bc3f60db01c6ca84828dec539','8983662f-86d1-41ab-8aad-a9c1c21d03dc',0,'2026-01-18 03:45:57','2026-01-18 03:45:57'),('895db1c0-ef9c-42d3-bae0-cc65980c81b9',19,'3634737.3661134.pdf',00000000000004519321,'4.31 MB',0,'pdf','https://go-cloud-store.oss-cn-shenzhen.aliyuncs.com/files/19/09ed07c9-43e5-47a6-9d96-9b64a8af7cad.pdf','https://go-cloud-store.oss-cn-shenzhen.aliyuncs.com/files/19/09ed07c9-43e5-47a6-9d96-9b64a8af7cad.pdf','files/19/09ed07c9-43e5-47a6-9d96-9b64a8af7cad.pdf','b9cc4789d854e4861d6845c4f7b6f0e8','19334b9d-f5c6-48a9-837a-d80db073858b',0,'2025-08-27 14:45:20','2025-08-27 14:45:20'),('8983662f-86d1-41ab-8aad-a9c1c21d03dc',20,'/',00000000000000000000,'',1,'','','','','',NULL,0,'2025-11-24 12:23:05','2025-11-24 12:23:04'),('89edd5e6-d617-47c4-bb8e-480326105fd0',8,'wallhaven-qzgjo7.png',00000000000003229887,'3.08 MB',0,'png','https://go-cloud-storage.oss-cn-beijing.aliyuncs.com/files/8/76a169e8-72b0-4d34-b809-16291d1ecd2a/89edd5e6-d617-47c4-bb8e-480326105fd0.png','https://go-cloud-storage.oss-cn-beijing.aliyuncs.com/files/8/76a169e8-72b0-4d34-b809-16291d1ecd2a/89edd5e6-d617-47c4-bb8e-480326105fd0.png','files/8/76a169e8-72b0-4d34-b809-16291d1ecd2a/89edd5e6-d617-47c4-bb8e-480326105fd0.png','5d9481c3d80672459bdc271d1438d615f8c95b6279a5326cf20242636d54d7e9','76a169e8-72b0-4d34-b809-16291d1ecd2a',0,'2025-08-14 09:16:51','2025-08-14 09:16:51'),('8b9ed35b-9199-4f4c-a88d-69e14865b0ff',18,'a5e82381bba6461783cd0c4fe1f3a128.png',00000000000000397734,'388.41 KB',0,'png','https://go-cloud-storage.oss-cn-beijing.aliyuncs.com/files/18/72915093-55c7-4ce3-a8fd-45830883ec53/8b9ed35b-9199-4f4c-a88d-69e14865b0ff.png','https://go-cloud-storage.oss-cn-beijing.aliyuncs.com/files/18/72915093-55c7-4ce3-a8fd-45830883ec53/8b9ed35b-9199-4f4c-a88d-69e14865b0ff.png','files/18/72915093-55c7-4ce3-a8fd-45830883ec53/8b9ed35b-9199-4f4c-a88d-69e14865b0ff.png','196853f21b2e7a36afabc3f13948ddc3a626de41bcacbdbb3f3fad1701f7e131','72915093-55c7-4ce3-a8fd-45830883ec53',0,'2025-08-19 06:11:51','2025-08-19 06:11:51'),('8ed5e5aaf78b85af50f4b5970ba6859c',19,'想见你.mp4',00000000000041624964,'39.70 MB',0,'mp4','https://go-cloud-store.oss-cn-shenzhen.aliyuncs.com/files/19/8ed5e5aaf78b85af50f4b5970ba6859c.mp4','https://go-cloud-store.oss-cn-shenzhen.aliyuncs.com/files/19/8ed5e5aaf78b85af50f4b5970ba6859c.mp4','files/19/8ed5e5aaf78b85af50f4b5970ba6859c.mp4','8ed5e5aaf78b85af50f4b5970ba6859c','19334b9d-f5c6-48a9-837a-d80db073858b',0,'2025-08-22 11:57:49','2025-08-22 11:57:49'),('9064ef4f-87ac-42e0-9a81-8c9eded32bac',18,'想见你.mp4',00000000000041624964,'39.70 MB',0,'mp4','https://go-cloud-storage.oss-cn-beijing.aliyuncs.com/files/18/72915093-55c7-4ce3-a8fd-45830883ec53/9064ef4f-87ac-42e0-9a81-8c9eded32bac.mp4','https://go-cloud-storage.oss-cn-beijing.aliyuncs.com/files/18/72915093-55c7-4ce3-a8fd-45830883ec53/9064ef4f-87ac-42e0-9a81-8c9eded32bac.mp4','files/18/72915093-55c7-4ce3-a8fd-45830883ec53/9064ef4f-87ac-42e0-9a81-8c9eded32bac.mp4','62190028eb62c74fff63948584ba6f23444ff0b826827748d83288b3f468a081','72915093-55c7-4ce3-a8fd-45830883ec53',0,'2025-08-19 06:13:44','2025-08-19 06:13:44'),('94f48de0-6a35-4e7a-9511-fd20fe7aa104',20,'wallhaven-vqy5p8.png',00000000000006360095,'6.07 MB',0,'png','http://192.168.101.129:9001/go-cloud-storage/files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/0328f9db-f833-4c73-88b8-d8bb124a5091.png','http://192.168.101.129:9001/go-cloud-storage/files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/0328f9db-f833-4c73-88b8-d8bb124a5091_thumb.jpg','files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/0328f9db-f833-4c73-88b8-d8bb124a5091.png','793d5688c7bf715acc355c2caddbc10d65c1c7ac9d0adc4b1a0d0ccbe9ac01cf','8983662f-86d1-41ab-8aad-a9c1c21d03dc',0,'2026-01-18 05:36:27','2026-01-18 05:36:27'),('974a1938-b9b3-4d58-b6c0-61245c5df104',20,'REAC T _ SYNERGIZING REASONING AND ACTING IN LANGUAGE MODELS.pdf',00000000000000512299,'500.29 KB',0,'pdf','http://192.168.101.129:9001/go-cloud-storage/files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/50d3b35f-0f6d-430c-a398-3ce4470a2d59.pdf','http://192.168.101.129:9001/go-cloud-storage/files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/50d3b35f-0f6d-430c-a398-3ce4470a2d59.pdf','files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/50d3b35f-0f6d-430c-a398-3ce4470a2d59.pdf','47759b51bacec12839ddb702c18542109de9cccfa3296c0cdc87690cca76c74a','8983662f-86d1-41ab-8aad-a9c1c21d03dc',0,'2026-01-18 03:35:19','2026-01-18 03:35:19'),('9850228263fb0650902665b98837e062',18,'Is It Just Me.mp4',00000000000037230272,'35.51 MB',0,'mp4','https://go-cloud-storage.oss-cn-beijing.aliyuncs.com/files/18/9850228263fb0650902665b98837e062','https://go-cloud-storage.oss-cn-beijing.aliyuncs.com/files/18/9850228263fb0650902665b98837e062','files/18/9850228263fb0650902665b98837e062','9850228263fb0650902665b98837e062','72915093-55c7-4ce3-a8fd-45830883ec53',0,'2025-08-21 06:42:00','2025-08-21 06:42:00'),('98b0757d-d73b-4a8a-9c2e-7205a6c869f3',17,'/',00000000000000000000,'',1,'','','','','',NULL,0,'2025-08-19 02:31:33','2025-08-19 02:31:32'),('9c837458-1eb9-4a73-801a-78e4ee31ec91',19,'【倚天村】图解数据结构(bugstack.cn 小傅哥).pdf',00000000000014112488,'13.46 MB',0,'pdf','https://go-cloud-store.oss-cn-shenzhen.aliyuncs.com/files/19/381df285-9f91-481c-ad8e-84774bf87f44.pdf','https://go-cloud-store.oss-cn-shenzhen.aliyuncs.com/files/19/381df285-9f91-481c-ad8e-84774bf87f44.pdf','files/19/381df285-9f91-481c-ad8e-84774bf87f44.pdf','fb8b1688673015dd0068456ea4d801cf','19334b9d-f5c6-48a9-837a-d80db073858b',0,'2025-08-27 13:57:33','2025-08-27 13:57:33'),('ad009b0e-399b-4404-98be-82e1ab66d843',19,'2024.lrec-main.1344.pdf',00000000000001016211,'992.39 KB',0,'pdf','https://go-cloud-store.oss-cn-shenzhen.aliyuncs.com/files/19/e239e2cb-c924-4545-8901-89e98d481fdf.pdf','https://go-cloud-store.oss-cn-shenzhen.aliyuncs.com/files/19/e239e2cb-c924-4545-8901-89e98d481fdf.pdf','files/19/e239e2cb-c924-4545-8901-89e98d481fdf.pdf','cde5e42414f05e547eb336a0d5acdd68','3fe885dc-6927-4147-a3c7-5f18b12e5edc',0,'2025-08-27 14:29:23','2025-08-27 14:29:23'),('aed5b808-b09f-4e92-96a7-59276b7459f5',20,'网安法3.pdf',00000000000000362754,'354.25 KB',0,'pdf','http://192.168.101.129:9001/go-cloud-storage/files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/b59f9ecc-9f63-47fe-b518-681b4d930661.pdf','http://192.168.101.129:9001/go-cloud-storage/files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/b59f9ecc-9f63-47fe-b518-681b4d930661.pdf','files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/b59f9ecc-9f63-47fe-b518-681b4d930661.pdf','13709aa4e44d580b0d5ea9fe12d1ba90f243cc998b0cd64cebe9fb318b7e5899','8983662f-86d1-41ab-8aad-a9c1c21d03dc',0,'2026-01-18 02:26:09','2026-01-18 02:26:09'),('b44c73a1-a35d-47e1-a314-ebcba3302e85',20,'8d003a30963b4ad0ba5bdb879d75f7ed.jpg',00000000000000653672,'638.35 KB',0,'jpg','http://192.168.101.129:9001/go-cloud-storage/files/20/78c1b61e-2dc2-4cbd-a3d7-71678741719a/390fae07-5662-4f7e-b965-693637911e7e.jpg','http://192.168.101.129:9001/go-cloud-storage/files/20/78c1b61e-2dc2-4cbd-a3d7-71678741719a/390fae07-5662-4f7e-b965-693637911e7e_thumb.jpg','files/20/78c1b61e-2dc2-4cbd-a3d7-71678741719a/390fae07-5662-4f7e-b965-693637911e7e.jpg','f98a04e5815b11ee0a01cdf5b556e63430cb953b0ec984d7be90068846811d1f','78c1b61e-2dc2-4cbd-a3d7-71678741719a',0,'2026-01-18 05:37:31','2026-01-18 05:37:31'),('b87b684f-6be1-48e9-9ced-88ec483978b3',19,'练手数据集.zip',00000000000047321091,'45.13 MB',0,'zip','https://go-cloud-store.oss-cn-shenzhen.aliyuncs.com/files/19/aca9df7f-1855-4e64-917a-da7ff584c5ac.zip','https://go-cloud-store.oss-cn-shenzhen.aliyuncs.com/files/19/aca9df7f-1855-4e64-917a-da7ff584c5ac.zip','files/19/aca9df7f-1855-4e64-917a-da7ff584c5ac.zip','7ba789dc484e04fa8c5be3e33b07212f','19334b9d-f5c6-48a9-837a-d80db073858b',0,'2025-08-24 07:49:08','2025-08-24 07:49:08'),('bcf1a96d-51c1-42d1-b41e-9a889bbf784d',8,'测试ACEF',00000000000000000000,'-',1,'','',NULL,'','','76a169e8-72b0-4d34-b809-16291d1ecd2a',0,'2025-08-08 08:48:37','2025-08-14 09:35:55'),('c03c61da-c064-419c-90e0-3c1259f4d489',5,'/',00000000000000000000,'-',1,'','',NULL,'','',NULL,0,'2025-08-01 15:11:33','2025-08-16 04:24:14'),('c6cff8b5-64b6-451a-8b47-4631850fc105',8,'2024.lrec-main.1344.pdf',00000000000001016211,'992.39 KB',0,'pdf','https://go-cloud-storage.oss-cn-beijing.aliyuncs.com/files/8/76a169e8-72b0-4d34-b809-16291d1ecd2a/c6cff8b5-64b6-451a-8b47-4631850fc105.pdf','https://go-cloud-storage.oss-cn-beijing.aliyuncs.com/files/8/76a169e8-72b0-4d34-b809-16291d1ecd2a/c6cff8b5-64b6-451a-8b47-4631850fc105.pdf','files/8/76a169e8-72b0-4d34-b809-16291d1ecd2a/c6cff8b5-64b6-451a-8b47-4631850fc105.pdf','97dbab97d7934e87c0f08b25b83e7e5a88af3ab8ed7a937e657799ba855c3d3d','76a169e8-72b0-4d34-b809-16291d1ecd2a',1,'2025-08-17 14:42:31','2025-08-17 15:12:57'),('c7ce1eec-bd08-44df-8f09-a95072b09778',20,'The Avengers_ A Simple Recipe for Uniting Smaller Language Models to Challenge Proprietary Giants.pdf',00000000000000803850,'785.01 KB',0,'pdf','http://192.168.101.129:9001/go-cloud-storage/files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/d77c69cf-f095-4f69-9e24-435d060595c1.pdf','http://192.168.101.129:9001/go-cloud-storage/files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/d77c69cf-f095-4f69-9e24-435d060595c1.pdf','files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/d77c69cf-f095-4f69-9e24-435d060595c1.pdf','f79c7ef772c059d6bde9121977778435968d8d267c903f308bfdbabf634c9b94','8983662f-86d1-41ab-8aad-a9c1c21d03dc',0,'2025-12-20 06:48:33','2025-12-20 08:03:44'),('c83b6471-34d6-4c4c-b50e-b080259a9a61',20,'mapreduce.pdf',00000000000000384452,'375.44 KB',0,'pdf','http://192.168.101.129:9001/go-cloud-storage/files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/56c8344f-f52c-4447-a95a-0641b9750ec0.pdf','http://192.168.101.129:9001/go-cloud-storage/files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/56c8344f-f52c-4447-a95a-0641b9750ec0.pdf','files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/56c8344f-f52c-4447-a95a-0641b9750ec0.pdf','9cfef3ef1b8fe1a1b66c7221f56c2eeca0b15d6608ea68b0c85a38bfbffd8ce5','8983662f-86d1-41ab-8aad-a9c1c21d03dc',0,'2026-01-18 02:13:52','2026-01-18 02:13:52'),('ce03f3ea-d62d-48a1-867e-9b2a6c0bba0b',8,'93e01746e49ebd2d53c6bc2d26e7c466.jpg',00000000000001230141,'1.17 MB',0,'jpg','https://go-cloud-storage.oss-cn-beijing.aliyuncs.com/files/8/bcf1a96d-51c1-42d1-b41e-9a889bbf784d/ce03f3ea-d62d-48a1-867e-9b2a6c0bba0b.jpg','https://go-cloud-storage.oss-cn-beijing.aliyuncs.com/files/8/bcf1a96d-51c1-42d1-b41e-9a889bbf784d/ce03f3ea-d62d-48a1-867e-9b2a6c0bba0b.jpg','files/8/bcf1a96d-51c1-42d1-b41e-9a889bbf784d/ce03f3ea-d62d-48a1-867e-9b2a6c0bba0b.jpg','c63b09a12e8115329ef50688ddba1fb791239bc02efc0cace4aab1c1b7c98bc8','bcf1a96d-51c1-42d1-b41e-9a889bbf784d',0,'2025-08-14 09:52:47','2025-08-14 09:52:47'),('cfbf48d4-e107-4c09-990e-e1c8a1bf0a6c',8,'我的资源',00000000000000000000,'-',1,'','',NULL,'','','76a169e8-72b0-4d34-b809-16291d1ecd2a',0,'2025-08-02 03:27:39','2025-08-14 09:36:04'),('d9749181-f32c-44a1-b7bd-8fd0d7f2dbc2',8,'测试BB',00000000000000000000,'-',1,'','','','','','bcf1a96d-51c1-42d1-b41e-9a889bbf784d',0,'2025-08-14 01:29:19','2025-08-14 09:36:07'),('e0285ad7-9469-418f-99e8-acfd452a5775',19,'MyBatisPlus.jpeg',00000000000000035407,'34.58 KB',0,'jpeg','https://go-cloud-store.oss-cn-shenzhen.aliyuncs.com/files/19/19334b9d-f5c6-48a9-837a-d80db073858b/e0285ad7-9469-418f-99e8-acfd452a5775.jpeg','https://go-cloud-store.oss-cn-shenzhen.aliyuncs.com/files/19/19334b9d-f5c6-48a9-837a-d80db073858b/e0285ad7-9469-418f-99e8-acfd452a5775.jpeg','files/19/19334b9d-f5c6-48a9-837a-d80db073858b/e0285ad7-9469-418f-99e8-acfd452a5775.jpeg','d81891f6bfaba7f28b7093a2f5277cf6b7158f47f1770203124d57c1919b4945','19334b9d-f5c6-48a9-837a-d80db073858b',0,'2025-08-22 06:49:00','2025-08-22 08:47:54'),('e5917142-c007-41e6-960b-77841e450b9e',8,'002237-166680135710f8.jpg',00000000000001181031,'1.13 MB',0,'jpg','https://go-cloud-storage.oss-cn-beijing.aliyuncs.com/files/8/76a169e8-72b0-4d34-b809-16291d1ecd2a/e5917142-c007-41e6-960b-77841e450b9e.jpg','https://go-cloud-storage.oss-cn-beijing.aliyuncs.com/files/8/76a169e8-72b0-4d34-b809-16291d1ecd2a/e5917142-c007-41e6-960b-77841e450b9e.jpg','files/8/76a169e8-72b0-4d34-b809-16291d1ecd2a/e5917142-c007-41e6-960b-77841e450b9e.jpg','11b1563ff5855abd59112d455b2b84744141d90fa10c9b0a08979cba25f89851','76a169e8-72b0-4d34-b809-16291d1ecd2a',1,'2025-08-14 10:25:52','2025-08-14 10:29:04'),('e72a05ab-4961-4fa8-b104-22a045562d55',20,'想见你.mp4',00000000000041624964,'39.70 MB',0,'mp4','http://192.168.101.129:9001/go-cloud-storage/files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/bc1c4657-e0fb-4cea-b5f3-4ec48531686d.mp4','http://192.168.101.129:9001/go-cloud-storage/files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/bc1c4657-e0fb-4cea-b5f3-4ec48531686d_thumb.jpg','files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/bc1c4657-e0fb-4cea-b5f3-4ec48531686d.mp4','62190028eb62c74fff63948584ba6f23444ff0b826827748d83288b3f468a081','8983662f-86d1-41ab-8aad-a9c1c21d03dc',0,'2025-12-15 11:52:10','2025-12-15 11:52:10'),('e9d71e76-4a51-42a8-b6ce-c5ccce8ee992',18,'64b790945d68323f21eb6d55f1b29b73.jpg',00000000000000956626,'934.21 KB',0,'jpg','https://go-cloud-storage.oss-cn-beijing.aliyuncs.com/files/18/72915093-55c7-4ce3-a8fd-45830883ec53/e9d71e76-4a51-42a8-b6ce-c5ccce8ee992.jpg','https://go-cloud-storage.oss-cn-beijing.aliyuncs.com/files/18/72915093-55c7-4ce3-a8fd-45830883ec53/e9d71e76-4a51-42a8-b6ce-c5ccce8ee992.jpg','files/18/72915093-55c7-4ce3-a8fd-45830883ec53/e9d71e76-4a51-42a8-b6ce-c5ccce8ee992.jpg','5038097642049c86d94d21dde6cfd0a7932b681144ecaad75d6d3ffcd3b51671','72915093-55c7-4ce3-a8fd-45830883ec53',0,'2025-08-19 04:01:50','2025-08-19 04:01:50'),('eaef214a-92d3-4fe6-b265-a41714370a98',20,'7318_Hybrid_LLM_Cost_Efficient.pdf',00000000000004420090,'4.22 MB',0,'pdf','http://192.168.101.129:9001/go-cloud-storage/files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/a808e9c8-4ce8-4040-b75a-7bc7f05fe84b.pdf','http://192.168.101.129:9001/go-cloud-storage/files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/a808e9c8-4ce8-4040-b75a-7bc7f05fe84b.pdf','files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/a808e9c8-4ce8-4040-b75a-7bc7f05fe84b.pdf','de4e956f77d538a283f96a5992088ad5c187ca8f59aa06aa6efad4f58af6a2ef','8983662f-86d1-41ab-8aad-a9c1c21d03dc',0,'2026-01-18 03:58:07','2026-01-18 03:58:07'),('ec97cd56-0f64-483c-aead-98476a2fef0f',19,'李知恩.jpg',00000000000000417835,'408.04 KB',0,'jpg','https://go-cloud-store.oss-cn-shenzhen.aliyuncs.com/files/19/19334b9d-f5c6-48a9-837a-d80db073858b/ec97cd56-0f64-483c-aead-98476a2fef0f.jpg','https://go-cloud-store.oss-cn-shenzhen.aliyuncs.com/files/19/19334b9d-f5c6-48a9-837a-d80db073858b/ec97cd56-0f64-483c-aead-98476a2fef0f.jpg','files/19/19334b9d-f5c6-48a9-837a-d80db073858b/ec97cd56-0f64-483c-aead-98476a2fef0f.jpg','c0c80a3f7c37ebbe97bd6849585985a348a3a20d370288dd5d2526b42ebdcf8f','19334b9d-f5c6-48a9-837a-d80db073858b',0,'2025-08-22 05:01:18','2025-08-22 08:47:32'),('ef7704ac-353b-4767-a55c-c1c3073cd590',20,'C卷真题考点.xlsx',00000000000000013229,'12.92 KB',0,'xlsx','http://192.168.101.129:9001/go-cloud-storage/files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/e877ffa2-858f-4d3c-8ab2-fd850c597080.xlsx','http://192.168.101.129:9001/go-cloud-storage/files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/e877ffa2-858f-4d3c-8ab2-fd850c597080.xlsx','files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/e877ffa2-858f-4d3c-8ab2-fd850c597080.xlsx','203d8f7071726ec2fa37b8d7bf7f203b0c7ac3be55807e7314cd7dec92c29ca7','8983662f-86d1-41ab-8aad-a9c1c21d03dc',0,'2025-12-20 09:50:54','2025-12-20 09:50:54'),('f0bc2089-5c23-45fe-91fd-f6681137ae52',8,'a.jpg',00000000000000978724,'955.79 KB',0,'jpg','https://go-cloud-storage.oss-cn-beijing.aliyuncs.com/files/8/76a169e8-72b0-4d34-b809-16291d1ecd2a/f0bc2089-5c23-45fe-91fd-f6681137ae52.jpg','https://go-cloud-storage.oss-cn-beijing.aliyuncs.com/files/8/76a169e8-72b0-4d34-b809-16291d1ecd2a/f0bc2089-5c23-45fe-91fd-f6681137ae52.jpg','files/8/76a169e8-72b0-4d34-b809-16291d1ecd2a/f0bc2089-5c23-45fe-91fd-f6681137ae52.jpg','2b3385671d5883b47489d19815f44b8f871af423e443d87bcd8bca4102fc4098','76a169e8-72b0-4d34-b809-16291d1ecd2a',0,'2025-08-14 10:22:09','2025-08-17 09:55:36'),('f5a09a2e-dd4a-4aad-b5d0-6e1ce9b36f5a',8,'鞠婧祎.jpg',00000000000001059082,'1.01 MB',0,'jpg','https://go-cloud-storage.oss-cn-beijing.aliyuncs.com/files/8/76a169e8-72b0-4d34-b809-16291d1ecd2a/f5a09a2e-dd4a-4aad-b5d0-6e1ce9b36f5a.jpg','https://go-cloud-storage.oss-cn-beijing.aliyuncs.com/files/8/76a169e8-72b0-4d34-b809-16291d1ecd2a/f5a09a2e-dd4a-4aad-b5d0-6e1ce9b36f5a.jpg','files/8/76a169e8-72b0-4d34-b809-16291d1ecd2a/f5a09a2e-dd4a-4aad-b5d0-6e1ce9b36f5a.jpg','fa781bc828cab12a50ab66ff194912acf5d172523f6d205d3ee498b4efa197a7','76a169e8-72b0-4d34-b809-16291d1ecd2a',0,'2025-08-14 09:52:12','2025-08-15 06:52:21'),('f795f744-0e16-4973-a69d-bab1e5d6181c',18,'003852-16792439327078.jpg',00000000000000464413,'453.53 KB',0,'jpg','https://go-cloud-storage.oss-cn-beijing.aliyuncs.com/files/18/72915093-55c7-4ce3-a8fd-45830883ec53/f795f744-0e16-4973-a69d-bab1e5d6181c.jpg','https://go-cloud-storage.oss-cn-beijing.aliyuncs.com/files/18/72915093-55c7-4ce3-a8fd-45830883ec53/f795f744-0e16-4973-a69d-bab1e5d6181c.jpg','files/18/72915093-55c7-4ce3-a8fd-45830883ec53/f795f744-0e16-4973-a69d-bab1e5d6181c.jpg','dbcac0f9ac9a82cd14d65c5a1cbcb700d5d05c107f2d28599785379b4a27d922','72915093-55c7-4ce3-a8fd-45830883ec53',0,'2025-08-19 06:11:05','2025-08-19 06:11:05'),('f7ab7d2c-0968-41b0-969e-a67361706a66',19,'PLeak-针对大型语言模型应用程序的即时泄漏攻击.pdf',00000000000002635664,'2.51 MB',0,'pdf','https://go-cloud-store.oss-cn-shenzhen.aliyuncs.com/files/19/1129f732-f060-452e-a070-3f8aaaf212e1/f7ab7d2c-0968-41b0-969e-a67361706a66.pdf','https://go-cloud-store.oss-cn-shenzhen.aliyuncs.com/files/19/1129f732-f060-452e-a070-3f8aaaf212e1/f7ab7d2c-0968-41b0-969e-a67361706a66.pdf','files/19/1129f732-f060-452e-a070-3f8aaaf212e1/f7ab7d2c-0968-41b0-969e-a67361706a66.pdf','026e6deb18150e5b7dd8b4a4f0dc271182fe2eee7e043778c5695d495d4abe31','1129f732-f060-452e-a070-3f8aaaf212e1',0,'2025-08-24 13:31:13','2025-08-24 13:31:13'),('f946149e-c834-4582-8cb8-90d3f451fa44',20,'wallhaven-5gzzp7.jpg',00000000000001728608,'1.65 MB',0,'jpg','http://192.168.101.129:9001/go-cloud-storage/files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/35645c42-f547-48ca-9f25-2400d6a26915.jpg','http://192.168.101.129:9001/go-cloud-storage/files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/35645c42-f547-48ca-9f25-2400d6a26915_thumb.jpg','files/20/8983662f-86d1-41ab-8aad-a9c1c21d03dc/35645c42-f547-48ca-9f25-2400d6a26915.jpg','ca974f33d956cd0a71ab82a874f962c752aaada258550426a6d77916e6d411b2','8983662f-86d1-41ab-8aad-a9c1c21d03dc',0,'2025-12-07 08:24:55','2025-12-07 08:24:55'),('fe279f86-8326-47e8-abe3-9154234a483a',19,'hymenoptera_data.zip',00000000000047286322,'45.10 MB',0,'zip','https://go-cloud-store.oss-cn-shenzhen.aliyuncs.com/files/19/a6b873d1-5a0b-40ca-94c9-982df2a6125a.zip','https://go-cloud-store.oss-cn-shenzhen.aliyuncs.com/files/19/a6b873d1-5a0b-40ca-94c9-982df2a6125a.zip','files/19/a6b873d1-5a0b-40ca-94c9-982df2a6125a.zip','5f8c32a6554f6acb4d649776e7735e48','19334b9d-f5c6-48a9-837a-d80db073858b',0,'2025-08-24 08:27:40','2025-08-24 08:27:40'),('ff52089c-6a2b-44c4-bb2a-16358f0050f7',18,'BigmodelPoster.png',00000000000000104268,'101.82 KB',0,'png','https://go-cloud-storage.oss-cn-beijing.aliyuncs.com/files/18/72915093-55c7-4ce3-a8fd-45830883ec53/ff52089c-6a2b-44c4-bb2a-16358f0050f7.png','https://go-cloud-storage.oss-cn-beijing.aliyuncs.com/files/18/72915093-55c7-4ce3-a8fd-45830883ec53/ff52089c-6a2b-44c4-bb2a-16358f0050f7.png','files/18/72915093-55c7-4ce3-a8fd-45830883ec53/ff52089c-6a2b-44c4-bb2a-16358f0050f7.png','17fb00844b60e816b3c0b900aa164d544029b6576adc11e1ae94af0d9cf43c70','72915093-55c7-4ce3-a8fd-45830883ec53',0,'2025-08-19 07:43:53','2025-09-08 04:45:16');
/*!40000 ALTER TABLE `file` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `notification`
--

DROP TABLE IF EXISTS `notification`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `notification` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` int NOT NULL,
  `title` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `message` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `type` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'info',
  `is_read` tinyint(1) NOT NULL DEFAULT '0',
  `link` varchar(500) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `idx_notifications_user_id` (`user_id`),
  KEY `idx_notifications_type` (`type`),
  KEY `idx_notifications_is_read` (`is_read`),
  KEY `idx_notifications_created_at` (`created_at`),
  CONSTRAINT `fk_notifications_user_id` FOREIGN KEY (`user_id`) REFERENCES `user` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='通知表';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `notification`
--

LOCK TABLES `notification` WRITE;
/*!40000 ALTER TABLE `notification` DISABLE KEYS */;
/*!40000 ALTER TABLE `notification` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `recycle_bin`
--

DROP TABLE IF EXISTS `recycle_bin`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `recycle_bin` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '回收站记录ID',
  `file_id` varchar(40) COLLATE utf8mb4_general_ci NOT NULL COMMENT '文件ID',
  `user_id` int NOT NULL COMMENT '所属用户ID',
  `deleted_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '删除时间',
  `expire_at` timestamp NOT NULL COMMENT '过期时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_user_file` (`user_id`,`file_id`),
  KEY `idx_expire_at` (`expire_at`),
  KEY `fk_recycle_file` (`file_id`),
  CONSTRAINT `fk_recycle_file` FOREIGN KEY (`file_id`) REFERENCES `file` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_recycle_user` FOREIGN KEY (`user_id`) REFERENCES `user` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=65 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `recycle_bin`
--

LOCK TABLES `recycle_bin` WRITE;
/*!40000 ALTER TABLE `recycle_bin` DISABLE KEYS */;
INSERT INTO `recycle_bin` VALUES (18,'56066a56-6271-40d9-bd89-c236f65d6753',8,'2025-08-14 09:31:35','2025-08-24 09:31:35'),(19,'e5917142-c007-41e6-960b-77841e450b9e',8,'2025-08-14 10:29:04','2025-08-24 10:29:04'),(20,'c6cff8b5-64b6-451a-8b47-4631850fc105',8,'2025-08-17 15:12:57','2025-08-27 15:12:57'),(39,'6cd0c92f-9ffe-4e25-a3e3-a6d231ecd2fb',19,'2025-11-22 13:31:06','2025-12-02 13:31:06');
/*!40000 ALTER TABLE `recycle_bin` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `share`
--

DROP TABLE IF EXISTS `share`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `share` (
  `id` int NOT NULL AUTO_INCREMENT,
  `user_id` int NOT NULL COMMENT '分享者ID',
  `file_id` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT '分享的文件/文件夹ID',
  `share_token` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT '唯一分享标识',
  `extraction_code` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '提取码',
  `expire_time` datetime DEFAULT NULL COMMENT '过期时间',
  `access_count` int DEFAULT '0' COMMENT '访问次数',
  `download_count` int DEFAULT '0' COMMENT '下载次数',
  `is_deleted` tinyint(1) DEFAULT '0' COMMENT '软删除标志',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_share_token` (`share_token`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_file_id` (`file_id`),
  CONSTRAINT `share_ibfk_1` FOREIGN KEY (`user_id`) REFERENCES `user` (`id`),
  CONSTRAINT `share_ibfk_2` FOREIGN KEY (`file_id`) REFERENCES `file` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=16 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='用户分享表';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `share`
--

LOCK TABLES `share` WRITE;
/*!40000 ALTER TABLE `share` DISABLE KEYS */;
INSERT INTO `share` VALUES (14,20,'5969825e-e8be-47c3-ad9f-055e27e4787e','c5d8052b4cb1a875cb0c672dd9a2e7bf',NULL,'2025-12-08 16:02:20',0,0,0,'2025-12-08 08:02:14','2025-12-08 08:02:20');
/*!40000 ALTER TABLE `share` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `storage_quota`
--

DROP TABLE IF EXISTS `storage_quota`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `storage_quota` (
  `user_id` int NOT NULL,
  `total` bigint DEFAULT '10737418240' COMMENT '总存储空间(字节);默认10GB',
  `used` bigint DEFAULT '0' COMMENT '已使用存储空间(字节)',
  `used_percent` float GENERATED ALWAYS AS (round(((`used` * 100.0) / `total`),2)) VIRTUAL COMMENT '使用百分比',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`user_id`),
  CONSTRAINT `storage_quota_ibfk_1` FOREIGN KEY (`user_id`) REFERENCES `user` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `storage_quota`
--

LOCK TABLES `storage_quota` WRITE;
/*!40000 ALTER TABLE `storage_quota` DISABLE KEYS */;
INSERT INTO `storage_quota` (`user_id`, `total`, `used`, `created_at`, `updated_at`) VALUES (8,10737418240,0,'2025-08-18 09:44:05','2025-08-18 09:44:06'),(18,10737418240,182550353,'2025-08-19 02:34:19','2025-08-21 07:51:36'),(19,10737418240,550447490,'2025-08-21 12:58:36','2025-11-22 13:31:06'),(20,10737418240,113137215,'2025-11-24 12:23:05','2026-01-18 05:40:38');
/*!40000 ALTER TABLE `storage_quota` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `upload_task`
--

DROP TABLE IF EXISTS `upload_task`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `upload_task` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `file_hash` varchar(64) NOT NULL COMMENT '文件唯一hash (SHA256/MD5)',
  `user_id` int NOT NULL COMMENT '用户ID',
  `file_name` varchar(255) NOT NULL COMMENT '原始文件名',
  `file_size` bigint NOT NULL COMMENT '字节大小',
  `total_chunks` int NOT NULL COMMENT '分片总数',
  `upload_id` varchar(255) DEFAULT NULL COMMENT 'OSS multipart uploadId',
  `status` tinyint NOT NULL DEFAULT '0' COMMENT '任务状态: 0-上传中, 1-已完成, 2-失败',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_file_user` (`file_hash`,`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='大文件上传任务表';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `upload_task`
--

LOCK TABLES `upload_task` WRITE;
/*!40000 ALTER TABLE `upload_task` DISABLE KEYS */;
/*!40000 ALTER TABLE `upload_task` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `user`
--

DROP TABLE IF EXISTS `user`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `user` (
  `id` int NOT NULL AUTO_INCREMENT,
  `username` varchar(50) COLLATE utf8mb4_general_ci NOT NULL,
  `email` varchar(100) COLLATE utf8mb4_general_ci NOT NULL,
  `phone` varchar(20) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `password` varchar(100) COLLATE utf8mb4_general_ci NOT NULL,
  `avatar` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `open_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `register_time` datetime(3) DEFAULT NULL,
  `root_folder_id` longtext COLLATE utf8mb4_general_ci,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uni_user_username` (`username`),
  UNIQUE KEY `uni_user_email` (`email`),
  UNIQUE KEY `uni_user_phone` (`phone`)
) ENGINE=InnoDB AUTO_INCREMENT=21 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `user`
--

LOCK TABLES `user` WRITE;
/*!40000 ALTER TABLE `user` DISABLE KEYS */;
INSERT INTO `user` VALUES (1,'User_22188772191885','2218877219@qq.com',NULL,'Cdq12345678','https://cube.elemecdn.com/3/7c/3ea6beec64369c2642b92c6726f1epng.png','','2025-07-10 10:57:18.000',NULL),(5,'User_18098269605410','1809826960@qq.com',NULL,'Abc123','https://cube.elemecdn.com/3/7c/3ea6beec64369c2642b92c6726f1epng.png','','2025-07-17 10:51:05.000','c03c61da-c064-419c-90e0-3c1259f4d489'),(6,'User_1234566458','123456@qq.com',NULL,'Abc123','https://cube.elemecdn.com/3/7c/3ea6beec64369c2642b92c6726f1epng.png','','2025-08-01 23:46:05.000',''),(8,'青祁','96119@qq.com','15290158828','Cdq123','https://go-cloud-storage.oss-cn-beijing.aliyuncs.com/avatars/8.jpg?t=1755144787','','2025-08-02 10:37:24.000','76a169e8-72b0-4d34-b809-16291d1ecd2a'),(11,'User_961188724','96118@qq.com',NULL,'Abc123','https://cube.elemecdn.com/3/7c/3ea6beec64369c2642b92c6726f1epng.png','','2025-08-18 21:44:37.000','7fdbe4ef-d387-49d6-a065-682ba7ae5abc'),(16,'User_123456909','12345@qq.com',NULL,'Cdq123','https://cube.elemecdn.com/3/7c/3ea6beec64369c2642b92c6726f1epng.png','','2025-08-19 10:25:25.000','82bb31dc-e48c-4c6b-837b-1db26f6286b7'),(17,'User_1233217363','123321@qq.com',NULL,'Cdq123','https://cube.elemecdn.com/3/7c/3ea6beec64369c2642b92c6726f1epng.png','','2025-08-19 10:31:33.000','98b0757d-d73b-4a8a-9c2e-7205a6c869f3'),(18,'大耳朵图图','666888@qq.com','13766668888','Cdq123','https://go-cloud-storage.oss-cn-beijing.aliyuncs.com/avatars/18.webp?t=1755574339','','2025-08-19 10:34:19.000','72915093-55c7-4ce3-a8fd-45830883ec53'),(19,'卡卡西','888999@qq.com','13566669999','Cdq123','http://192.168.101.129:9001/go-cloud-storage/avatars/19.jpg?t=1763801364','','2025-08-21 20:58:36.000','19334b9d-f5c6-48a9-837a-d80db073858b'),(20,'User_abc1238254','abc123@qq.com',NULL,'Cdq123','http://192.168.101.129:9001/go-cloud-storage/avatars/20.jpg?t=1763991200','','2025-11-24 20:23:04.619','8983662f-86d1-41ab-8aad-a9c1c21d03dc');
/*!40000 ALTER TABLE `user` ENABLE KEYS */;
UNLOCK TABLES;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2026-01-24 14:54:54
